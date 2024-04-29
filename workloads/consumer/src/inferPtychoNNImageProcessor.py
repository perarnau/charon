import os
import numpy as np
import multiprocessing as mp
import subprocess
import threading
import queue
import time
import psutil
import pvapy as pva
from pvapy.hpc.adImageProcessor import AdImageProcessor

class InferPtychoNNImageProcessor(AdImageProcessor):

    def __init__(self, configDict={}):
        AdImageProcessor.__init__(self, configDict)
        self.tq_frame_q = mp.Queue(maxsize=-1)
        self.batch_q = mp.Queue(maxsize=-1)
        self.frm_id_q = mp.Queue(maxsize=-1)
        self.nFramesProcessed = 0
        self.nBatchesProcessed = 0
        self.inferTime = 0

        self.bsz = configDict.get('bsz', 8)
        self.onnx_mdl = configDict.get('onnx_mdl', '/app/model_128.trt')
        self.output_x = configDict.get("output_x", 128)
        self.output_y = configDict.get("output_y", 128)
        self.nGPU = int(configDict.get("n_gpu", 1))

        # monitoring
        self.network_interface_to_monitor = configDict.get("net", "")
        self.m_network_rx_last = 0.
        self.m_network_tx_last = 0.
        self.isDone = False

    def inferWorker(self):
        self.logger.debug('Starting infer worker')
        self.gpu = (self.processorId - 1) % self.nGPU
        self.logger.debug(f'Using gpu: {self.gpu}')
        os.environ['CUDA_VISIBLE_DEVICES'] = str(self.gpu)
        #from pvaInferPtychoNN import inferPtychoNNtrt
        from inferPtychoNN import inferPtychoNNtrt
        self.inferEngine = inferPtychoNNtrt(self, mbsz=self.bsz, onnx_mdl=self.onnx_mdl, tq_diff=self.batch_q, frm_id_q=self.frm_id_q)
        self.logger.debug(f'Created infer engine using mbsz={self.bsz} and onnx_mdl={self.onnx_mdl}')
        bsz = self.bsz
        batch_list =[]
        frm_id_list = []

        waitTime = 1
        while True:
            if self.isDone:
                break
            try:
                frm_id, in_frame, ny, nx, attr = self.tq_frame_q.get(block=True, timeout=waitTime)
            except queue.Empty:
                continue
            except KeyboardInterrupt:
                self.isDone = True
                break
            except EOFError:
                self.isDone = True
                break
            except Exception as ex:
                self.logger.error(f'Unexpected error caught: {ex} {type(ex)}')
                break

            batch_list.append(in_frame)
            frm_id_list.append(frm_id)

            while(len(batch_list)>=bsz and not self.isDone):
                batch_chunk = (np.array(batch_list[:bsz]).astype(np.float32))
                batch_frm_id = np.array((frm_id_list[:bsz]))
                self.batch_q.put(batch_chunk)
                self.frm_id_q.put(batch_frm_id)
                batch_list=batch_list[bsz:]
                frm_id_list = frm_id_list[bsz:]

                t0 = time.time()
                self.inferEngine.batch_infer(nx, ny, self.output_x, self.output_y, attr)
                t1 = time.time()
                self.nBatchesProcessed += 1
                self.nFramesProcessed += bsz
                self.inferTime += t1-t0

        try:
            self.logger.debug(f'Stopping infer engine')
            self.inferEngine.stop()
        except Exception as ex:
            self.logger.warn(f'Error stopping infer engine: {ex}')
        self.logger.debug('Infer worker is done')

    def start(self):
        self.inferThread = threading.Thread(target=self.inferWorker)
        self.inferThread.start()

    def stop(self):
        self.logger.debug('Signaling infer worker to stop')
        self.isDone = True

    def configure(self, kwargs):
        self.logger.debug(f'Configuration update: {kwargs}')

    def process(self, pvObject):
        if self.isDone:
            return
        (frameId,image,nx,ny,nz,colorMode,fieldKey) = self.reshapeNtNdArray(pvObject)
        attributes = []
        if 'attribute' in pvObject:
            attributes = pvObject['attribute']
        self.tq_frame_q.put((frameId, image, ny, nx, attributes))
        return pvObject

    def resetStats(self):
        self.nFramesProcessed = 0
        self.nBatchesProcessed = 0
        self.inferTime = 0
        self.m_network_tx_last = 0.
        self.m_network_rx_last = 0.

    def getMetrics(self):
        m = {}
        g_metric_command = "/usr/bin/nvidia-smi --query-gpu=memory.used,power.draw,utilization.gpu,utilization.memory --format=csv,noheader,nounits"
        # we expect to get a csv looking like,
        # b'877, 10.98, 0, 0\n'
        g_metric_raw = subprocess.check_output(g_metric_command, shell=True).decode().strip()
        sp = g_metric_raw.split(",")
        m["gpuMemUsed"] = int(sp[0].strip())
        m["gpuPowerWUsed"] = float(sp[1].strip())
        m["gpuUtil"] = int(sp[2].strip())
        m["gpuUtilMem"] = int(sp[3].strip())
        # according to https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt
        # total used memory = RSS + CACHE + (SWAP)
        # we ingore SWAP as we don't expect to use it
        with open("/sys/fs/cgroup/memory/memory.stat") as file:
            cache = 0
            rss = 0
            for l in file:
                sp = l.split(" ")
                if sp[0] == "cache":
                    cache = int(sp[1])
                elif sp[0] == "rss":
                    rss = int(sp[1])
            # make it to GB
            m["memUsed"] = (rss + cache) / 1e9
        m["cpuUtil"] = float(psutil.cpu_percent())
        if self.network_interface_to_monitor != "":
            n = psutil.net_io_counters(pernic=True)
            if self.network_interface_to_monitor not in n:
                self.logger.error(f'failed to find network {self.network_interface_to_monitor} to monitor')
            else:
                # expected format is
                # snetio(bytes_sent=8670463628, bytes_recv=7493883175, packets_sent=30910313, packets_recv=20757489, errin=1, errout=0, dropin=0, dropout=0)
                n_metric = list(n[self.network_interface_to_monitor])
                # make them to GB
                tx = n_metric[0]/1e9
                rx = n_metric[1]/1e9
                # the first measurement skips reporting
                if self.m_network_rx_last == 0.:
                    m["netTxUsed"] = 0.
                    m["netRxUsed"] = 0.
                    self.m_network_tx_last = tx
                    self.m_network_rx_last = rx
                else:
                    m["netTxUsed"] = tx - self.m_network_tx_last
                    m["netRxUsed"] = rx - self.m_network_rx_last
                    self.m_network_tx_last = tx
                    self.m_network_rx_last = rx
        return m


    # Retrieve statistics for user processor
    def getStats(self):
        stat = self.getMetrics()
        inferRate = 0
        frameProcessingRate = 0
        if self.nBatchesProcessed  > 0:
            inferRate = self.nBatchesProcessed /self.inferTime
            frameProcessingRate = self.nFramesProcessed/self.inferTime
        nFramesQueued = self.tq_frame_q.qsize()
        stat.update({
            'nFramesProcessed' : self.nFramesProcessed,
            'nBatchesProcessed' : self.nBatchesProcessed,
            'nFramesQueued' : nFramesQueued,
            'inferTime' : self.inferTime,
            'inferRate' : inferRate,
            'frameProcessingRate' : frameProcessingRate
        })
        return stat

    # Define PVA types for different stats variables
    def getStatsPvaTypes(self):
        return {
            "gpuMemUsed": pva.UINT,
            "gpuPowerWUsed": pva.FLOAT,
            "gpuUtil": pva.UINT,
            "gpuUtilMem": pva.UINT,
            "memUsed": pva.FLOAT,
            "cpuUtil": pva.FLOAT,
            "netTxUsed": pva.FLOAT,
            "netRxUsed": pva.FLOAT,
            'nFramesProcessed' : pva.UINT,
            'nBatchesProcessed' : pva.UINT,
            'nFramesQueued' : pva.UINT,
            'inferTime' : pva.DOUBLE,
            'inferRate' : pva.DOUBLE,
            'frameProcessingRate' : pva.DOUBLE
        }

    def getOutputPvObjectType(self, pvObject):
        return pva.NtNdArray()
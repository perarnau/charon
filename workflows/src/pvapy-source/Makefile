docker:
	docker build -t local/pvapy-udsource:0.0.1 --load .

k3s-import:
	docker save -o /tmp/pvapy-udsource.tar local/pvapy-udsource:0.0.1
	k3s ctr image import /tmp/pvapy-udsource.tar
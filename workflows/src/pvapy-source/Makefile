build:
	docker build -t local/pvapy-udsource:0.0.2 --load .

buildx:
	docker buildx build -t local/pvapy-udsource:0.0.2 --load .

k3s-import:
	docker save -o /tmp/pvapy-udsource.tar local/pvapy-udsource:0.0.2
	k3s ctr image import /tmp/pvapy-udsource.tar
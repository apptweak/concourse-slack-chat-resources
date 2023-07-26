
all: read-resource post-resource

read-resource:
	docker build --platform "linux/x86_64" -t apptweakci/slack-read-resource -f read/Dockerfile .

post-resource:
	docker build --platform "linux/x86_64" -t apptweakci/slack-post-resource -f post/Dockerfile .

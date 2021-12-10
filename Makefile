
all: read-resource post-resource

read-resource:
	docker build -t apptweakci/slack-read-resource -f read/Dockerfile .

post-resource:
	docker build -t apptweakci/slack-post-resource -f post/Dockerfile .

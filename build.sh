#Pass version using command line argument
command sudo docker build --no-cache  --platform linux/amd64 -t dafraer/create-video-note-bot:$1 .
command  docker push dafraer/create-video-note-bot:$1
#Run commands for raspbrry pi
#Pass token as an argument
command sudo docker build --no-cache  -t create-video-note-bot .
command sudo docker run -d -e TOKEN=$1 -e WEBHOOK=false create-video-note-bot
#Pass token using command line argument
command sudo docker build --no-cache -t create-video-note-bot .
command sudo docker run -d -e TOKEN=$1 create-video-note-bot
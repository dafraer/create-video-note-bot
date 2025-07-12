#Pass version using command line argument
command sudo docker build --no-cache -t create-video-note-bot:$1 .
command sudo docker run -d -e TOKEN=8032716817:AAE0JEthFKACO3ey_EWSrtSLmfBkOchZQdA create-video-note-bot:$1
sudo docker build -t eclipse-vnc .
sudo docker run -p 5901:5901 eclipse-vnc
sudo docker stop $(sudo docker ps -a -q)
sudo docker rm $(sudo docker ps -a -q)
sudo docker builder prune -a
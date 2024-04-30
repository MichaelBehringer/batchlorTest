docker build -t eclipse-vnc .
docker run -d -p 5901:5901 eclipse-vnc

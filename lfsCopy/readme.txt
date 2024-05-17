podman build -t lfs-vnc .

podman stop $(podman ps -a -q)
podman rm $(podman ps -a -q)
podman rmi $(podman images -qa) -f

buildah bud --layers --network host --tag=lfs-vnc -f Dockerfile . 
podman run -it --network host --name lfsx-web-lfs -u 1111:1001 --cap-drop ALL -p 5910:5910 lfs-vnc
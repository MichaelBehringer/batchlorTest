## Gitea Secret

To build this docker image you have to create a docker secret first with your Gitea API-key that allows access the Repository`hama.java.lfs`. You can create an API-Key [here](https://gitea.hama.de/user/settings/applications) for Gitea.  
Put the created API-Key inside a raw text file. Now you can create the secret.

```
 podman secret create giteaApiKey /path/to/apiKey
```

## Build

```
buildah bud --layers --network host --tag=hama.de/lfsx-web-lfs:v0 --secret id=giteaApiKey,src=/home/ubuntugui/.secrets/gitea_api-key
```

## Running

To run the container with the same permissions as in OpenShfit you can execute the following command.

```
podman run --network host -u 1111:1001 --cap-drop ALL -p 5910:5910 hama.de/lfsx-web-lfs:0.0.0
```
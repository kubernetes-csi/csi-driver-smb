# How to build cross-platform container images

```console
export DOCKER_CLI_EXPERIMENTAL=enabled

acrName=
az acr login -n $acrName

acrRepo=$acrName.azurecr.io/public/k8s/csi/smb-csi
ver=v0.6.0

linux="linux-amd64"
make smb
az acr build -r $acrName -t $acrRepo:$ver-$linux -f cmd/smbplugin/Dockerfile  --platform linux .

win="windows-1809-amd64"
make smb-windows
az acr build -r $acrName -t $acrRepo:$ver-$win -f cmd/smbplugin/Dockerfile.Windows --platform windows .

docker manifest create $acrRepo:$ver $acrRepo:$ver-$linux $acrRepo:$ver-$win
docker manifest inspect $acrRepo:$ver
docker manifest push $acrRepo:$ver --purge

docker manifest create $acrRepo:latest $acrRepo:$ver-$linux $acrRepo:$ver-$win
docker manifest inspect $acrRepo:latest
docker manifest push $acrRepo:latest --purge

# check
docker manifest inspect mcr.microsoft.com/k8s/csi/smb-csi:$ver
docker manifest inspect mcr.microsoft.com/k8s/csi/smb-csi:latest
```

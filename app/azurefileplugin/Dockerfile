FROM alpine
LABEL maintainers="andyzhangx"
LABEL description="AzureFile CSI Driver"

COPY ./_output/azurefileplugin /azurefileplugin
ENTRYPOINT ["/azurefileplugin"]

FROM ubuntu:16.04
RUN apt-get update && apt-get install -y build-essential ca-certificates cifs-utils
LABEL maintainers="andyzhangx"
LABEL description="AzureFile CSI Driver"

COPY ./_output/azurefileplugin /azurefileplugin
ENTRYPOINT ["/azurefileplugin"]

FROM debian:9
RUN apt-get update && apt-get install -y ca-certificates cifs-utils
LABEL maintainers="andyzhangx"
LABEL description="AzureFile CSI Driver"

COPY ./_output/azurefileplugin /azurefileplugin
ENTRYPOINT ["/azurefileplugin"]

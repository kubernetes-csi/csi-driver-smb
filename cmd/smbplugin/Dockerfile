# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM debian:bookworm AS cifs-utils-builder

ARG CIFS_UTILS_VERSION=7.0
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    bzip2 \
    build-essential \
    ca-certificates \
    curl \
    libcap-ng-dev \
    libkeyutils-dev \
    libkrb5-dev \
    libpam0g-dev \
    libtalloc-dev \
    libwbclient-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN curl -fsSLo cifs-utils.tar.bz2 https://www.samba.org/ftp/linux-cifs/cifs-utils/cifs-utils-${CIFS_UTILS_VERSION}.tar.bz2 \
    && tar -xjf cifs-utils.tar.bz2 --strip-components=1

RUN multiarch="$(gcc -print-multiarch)" \
    && ./configure \
        --enable-cifsidmap \
        --enable-cifscreds \
        --with-libcap-ng=auto \
        --enable-pam \
        --with-pamdir="/lib/${multiarch}/security" \
        --enable-man=no \
        --enable-smbinfo=no \
        --disable-pythontools \
    && make -j"$(nproc)" \
    && make DESTDIR=/out install \
    && install -d /out/etc/request-key.d /out/etc/cifs-utils \
    && printf 'create  cifs.idmap    * * /usr/sbin/cifs.idmap %%k\n' > /out/etc/request-key.d/cifs.idmap.conf \
    && printf 'create  cifs.spnego    * * /usr/sbin/cifs.upcall %%k\n' > /out/etc/request-key.d/cifs.spnego.conf \
    && ln -sf "/usr/lib/${multiarch}/cifs-utils/idmapwb.so" /out/etc/cifs-utils/idmap-plugin \
    && chmod u+s /out/sbin/mount.cifs

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.8

RUN apt update && apt upgrade -y && apt-mark unhold libcap2 && clean-install \
    ca-certificates \
    e2fsprogs \
    libcap-ng0 \
    libgssapi-krb5-2 \
    libkeyutils1 \
    libkrb5-3 \
    libpam0g \
    libtalloc2 \
    libwbclient0 \
    mount \
    udev \
    util-linux \
    xfsprogs

COPY --from=cifs-utils-builder /out/ /

LABEL maintainers="andyzhangx"
LABEL description="SMB CSI Driver"
ARG ARCH=amd64
ARG binary=./_output/${ARCH}/smbplugin
COPY ${binary} /smbplugin

RUN echo "[libdefaults]\n    default_ccache_name = FILE:/var/lib/kubelet/kerberos/krb5cc_%{uid}\n" > /etc/krb5.conf

ENTRYPOINT ["/smbplugin"]

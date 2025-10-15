ARG BASE_IMAGE="nvcr.io/nvidia/cuda"
ARG BASE_IMAGE_TAG="12.5.1-devel-ubuntu24.04"

ARG RUNTIME_IMAGE="nvcr.io/nvidia/cuda"
ARG RUNTIME_IMAGE_TAG="12.5.1-devel-ubuntu24.04"

FROM ${BASE_IMAGE}:${BASE_IMAGE_TAG} AS nixl_base
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -y && apt-get install git -y

# ##################################
# ########## Build Image ###########
# ##################################

FROM ${BASE_IMAGE}:${BASE_IMAGE_TAG} AS build
ENV DEBIAN_FRONTEND=noninteractive

USER root

### NIXL Dependencies setup ###

ARG MOFED_VERSION=24.10-1.1.4.0
ARG PYTHON_VERSION=3.12
ARG NSYS_URL=https://developer.nvidia.com/downloads/assets/tools/secure/nsight-systems/2025_1/
ARG NSYS_PKG=NsightSystems-linux-cli-public-2025.1.1.131-3554042.deb

RUN apt-get update -y && apt-get -y install curl \
    git \
    libnuma-dev \
    numactl \
    wget \
    autotools-dev \
    automake \
    libtool \
    libz-dev \
    libiberty-dev \
    flex \
    build-essential \
    cmake \
    libibverbs-dev \
    libgoogle-glog-dev \
    libgtest-dev \
    libjsoncpp-dev \
    libpython3-dev \
    libboost-all-dev \
    libssl-dev \
    libgrpc-dev \
    libgrpc++-dev \
    libprotobuf-dev \
    libclang-dev \
    protobuf-compiler-grpc \
    pybind11-dev \
    python3-full \
    python3-pip \
    python3-numpy \
    etcd-server \
    net-tools \
    pciutils \
    libpci-dev \
    vim \
    tmux \
    screen \
    ibverbs-utils \
    libibmad-dev

RUN apt-get install -y linux-tools-common linux-tools-generic ethtool iproute2
RUN apt-get install -y dkms linux-headers-generic
RUN apt-get install -y meson ninja-build uuid-dev gdb

RUN apt install -y libglib2.0-0
RUN wget ${NSYS_URL}${NSYS_PKG} &&\
    apt install -y ./${NSYS_PKG} &&\
    rm ${NSYS_PKG}

RUN cd /usr/local/src && \
    curl -fSsL "https://content.mellanox.com/ofed/MLNX_OFED-${MOFED_VERSION}/MLNX_OFED_LINUX-${MOFED_VERSION}-ubuntu24.04-x86_64.tgz" -o mofed.tgz && \
    tar -xf /usr/local/src/mofed.tgz && \
    cd MLNX_OFED_LINUX-* && \
    apt-get update && apt-get install -y --no-install-recommends \
    ./DEBS/libibverbs* ./DEBS/ibverbs-providers* ./DEBS/librdmacm* ./DEBS/libibumad* && \
    rm -rf /var/lib/apt/lists/* /usr/local/src/* mofed.tgz

ENV LIBRARY_PATH=$LIBRARY_PATH:/usr/local/cuda/lib64 \
    LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/cuda/lib64

ENV LIBRARY_PATH=$LIBRARY_PATH:/usr/local/lib \
    LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib

WORKDIR /workspace
RUN git clone https://github.com/NVIDIA/gdrcopy.git
RUN PREFIX=/usr/local DESTLIB=/usr/local/lib make -C /workspace/gdrcopy lib_install
RUN cp gdrcopy/src/libgdrapi.so.2.* /usr/lib/x86_64-linux-gnu/
RUN ldconfig

ARG UCX_VERSION=v1.18.0

RUN cd /usr/local/src && \
    curl -fSsL "https://github.com/openucx/ucx/tarball/${UCX_VERSION}" | tar xz && \
    cd openucx-ucx* && \
    ./autogen.sh && ./configure     \
    --enable-shared             \
    --disable-static            \
    --disable-doxygen-doc       \
    --enable-optimizations      \
    --enable-cma                \
    --enable-devel-headers      \
    --with-cuda=/usr/local/cuda \
    --with-verbs                \
    --with-dm                   \
    --with-gdrcopy=/usr/local   \
    --enable-mt                 \
    --with-mlx5-dv &&           \
    make -j &&                      \
    make -j install-strip &&        \
    ldconfig

ENV LD_LIBRARY_PATH=/usr/lib:$LD_LIBRARY_PATH
ENV CPATH=/usr/include:$CPATH
ENV PATH=/usr/bin:$PATH
ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig:$PKG_CONFIG_PATH
SHELL ["/bin/bash", "-c"]

WORKDIR /workspace

ENV LD_LIBRARY_PATH=/usr/local/ompi/lib:$LD_LIBRARY_PATH
ENV CPATH=/usr/local/ompi/include:$CPATH
ENV PATH=/usr/local/ompi/bin:$PATH
ENV PKG_CONFIG_PATH=/usr/local/ompi/lib/pkgconfig:$PKG_CONFIG_PATH

### NIXL setup ###

WORKDIR /opt
RUN mkdir -p /opt/nixl
WORKDIR /opt/nixl

ARG NIXL_VERSION="0.1.1"

RUN wget "https://github.com/ai-dynamo/nixl/archive/refs/tags/${NIXL_VERSION}.tar.gz"
RUN tar --strip-components=1 -zxvf 0.1.1.tar.gz; rm 0.1.1.tar.gz


RUN cd /opt/nixl && \
    mkdir build && \
    meson setup build/ --prefix=/usr/local/nixl && \
    cd build/ && \
    ninja && \
    ninja install

ENV LD_LIBRARY_PATH=/usr/local/nixl/lib/x86_64-linux-gnu/:$LD_LIBRARY_PATH
ENV PYTHONPATH=/usr/local/nixl/lib/python3/dist-packages/:/opt/nixl/test/python/:$PYTHONPATH
ENV NIXL_PLUGIN_DIR=/usr/local/nixl/lib/x86_64-linux-gnu/plugins

## for benchmarking ##
WORKDIR /workspace

RUN python3 -m venv venv
ENV PATH="/workspace/venv/bin:$PATH"

RUN pip3 install torch zmq numpy

COPY benchmarks/benchmark.py benchmark.py


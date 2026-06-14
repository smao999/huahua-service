FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    grpcio>=1.60.0 \
    grpcio-tools>=1.60.0 \
    akshare>=1.15.0 \
    pandas>=2.0.0

# 生成 Python gRPC stub（从 proto 文件）
COPY akshare_proto/akshare.proto ./
RUN python3 -m grpc_tools.protoc -I. \
    --python_out=. \
    --grpc_python_out=. \
    akshare.proto \
    && rm akshare.proto

COPY akshare_server/server.py .

EXPOSE 9800

ENTRYPOINT ["python3", "server.py"]

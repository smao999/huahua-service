"""
akshare gRPC Server —— 将 akshare 库暴露为 gRPC 服务

启动方式:
    python3 akshare_server/server.py

端口: localhost:9800
"""

import json
import logging
import sys
import traceback
from concurrent import futures

import grpc

try:
    import akshare as ak
except ImportError:
    print("[FATAL] akshare 未安装，请执行: python3 -m pip install akshare")
    sys.exit(1)

try:
    import pandas as pd
except ImportError:
    pd = None

import akshare_pb2
import akshare_pb2_grpc

logging.basicConfig(level=logging.INFO, format="[akshare-server] %(asctime)s %(message)s")
logger = logging.getLogger(__name__)


def _serialize(result) -> str:
    """将 akshare 返回值转为 JSON 字符串。"""
    if pd is not None and isinstance(result, pd.DataFrame):
        return result.to_json(orient="records", force_ascii=False)
    return json.dumps(result, ensure_ascii=False, default=str)


class AkshareServicer(akshare_pb2_grpc.AkshareServicer):
    """gRPC 服务实现。"""

    def Call(self, request, context):
        fn_name = request.fn
        kwargs = dict(request.kwargs)
        logger.info("Call: %s(%s)", fn_name, kwargs)

        try:
            fn = getattr(ak, fn_name, None)
            if fn is None:
                return akshare_pb2.CallResponse(
                    ok=False,
                    error=f"akshare 中没有函数: {fn_name}",
                )

            result = fn(**kwargs)
            data_json = _serialize(result)
            return akshare_pb2.CallResponse(ok=True, data_json=data_json)

        except Exception as e:
            logger.error("Call 异常: %s", traceback.format_exc())
            return akshare_pb2.CallResponse(ok=False, error=str(e))

    def BatchGetFundEstimates(self, request, context):
        codes = list(request.codes)
        logger.info("BatchGetFundEstimates: %d codes", len(codes))

        estimates = []
        for code in codes:
            try:
                result = ak.fund_open_fund_info_em(symbol=code, indicator="单位净值走势")
                if pd is not None and isinstance(result, pd.DataFrame):
                    record = result.iloc[-1].to_dict() if len(result) > 0 else {}
                else:
                    record = result if isinstance(result, dict) else {}

                estimates.append(akshare_pb2.FundEstimate(
                    fund_code=code,
                    name=record.get("基金简称", ""),
                    estimate=float(record.get("单位净值", 0) or 0),
                    estimate_percent=float(record.get("日增长率", 0) or 0),
                    last_nav=0.0,
                    estimate_time="",
                ))
            except Exception as e:
                logger.warning("获取估值失败 %s: %s", code, e)

        return akshare_pb2.BatchEstimateResponse(estimates=estimates)

    def GetFundHistory(self, request, context):
        code = request.code
        logger.info("GetFundHistory: %s", code)

        try:
            result = ak.fund_open_fund_info_em(symbol=code, indicator="单位净值走势")

            records = []
            if pd is not None and isinstance(result, pd.DataFrame):
                for _, row in result.iterrows():
                    records.append(akshare_pb2.NavRecord(
                        date=str(row.get("净值日期", "")),
                        nav=float(row.get("单位净值", 0) or 0),
                        change=float(row.get("日增长率", 0) or 0),
                    ))

            return akshare_pb2.HistoryResponse(records=records)

        except Exception as e:
            logger.error("GetFundHistory 异常: %s", traceback.format_exc())
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return akshare_pb2.HistoryResponse()


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    akshare_pb2_grpc.add_AkshareServicer_to_server(AkshareServicer(), server)
    server.add_insecure_port("[::]:9800")
    server.start()
    logger.info("akshare gRPC server 已启动: :9800")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()

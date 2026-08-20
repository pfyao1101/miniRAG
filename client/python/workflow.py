import argparse
import logging
import math

import grpc

from minirag.v1 import storage_pb2
from minirag.v1 import storage_pb2_grpc


DEFAULT_TARGET = "127.0.0.1:50051"
DEFAULT_TIMEOUT = 2.0

RECORDS = (
    storage_pb2.Record(
        id="b",
        vector=[1, 0],
        text="record b",
        metadata={"source": "test"},
    ),
    storage_pb2.Record(
        id="a",
        vector=[1, 0],
        text="record a",
        metadata={"source": "test"},
    ),
    storage_pb2.Record(
        id="c",
        vector=[0, 1],
        text="record c",
        metadata={"source": "test"},
    ),
)


def delete_if_exists(
    stub: storage_pb2_grpc.VectorStoreServiceStub,
    record_id: str,
    timeout: float,
) -> None:
    try:
        stub.Delete(
            storage_pb2.DeleteRequest(id=record_id),
            timeout=timeout,
        )
    except grpc.RpcError as error:
        if error.code() == grpc.StatusCode.NOT_FOUND:
            return
        raise


def cleanup_records(
    stub: storage_pb2_grpc.VectorStoreServiceStub,
    timeout: float,
) -> None:
    for record in RECORDS:
        delete_if_exists(stub, record.id, timeout)


def insert_records(
    stub: storage_pb2_grpc.VectorStoreServiceStub,
    timeout: float,
) -> None:
    for record in RECORDS:
        stub.Insert(
            storage_pb2.InsertRequest(record=record),
            timeout=timeout,
        )


def get_and_validate(
    stub: storage_pb2_grpc.VectorStoreServiceStub,
    expected: storage_pb2.Record,
    timeout: float,
) -> None:
    response = stub.Get(
        storage_pb2.GetRequest(id=expected.id),
        timeout=timeout,
    )
    actual = response.record

    if actual.id != expected.id:
        raise ValueError(f"Get({expected.id!r}) id = {actual.id!r}")
    if list(actual.vector) != list(expected.vector):
        raise ValueError(
            f"Get({expected.id!r}) vector = {list(actual.vector)!r}, "
            f"want {list(expected.vector)!r}"
        )
    if actual.text != expected.text:
        raise ValueError(
            f"Get({expected.id!r}) text = {actual.text!r}, want {expected.text!r}"
        )
    if dict(actual.metadata) != dict(expected.metadata):
        raise ValueError(
            f"Get({expected.id!r}) metadata = {dict(actual.metadata)!r}, "
            f"want {dict(expected.metadata)!r}"
        )


def search_and_validate(
    stub: storage_pb2_grpc.VectorStoreServiceStub,
    timeout: float,
) -> None:
    response = stub.Search(
        storage_pb2.SearchRequest(query=[1, 0], k=2),
        timeout=timeout,
    )

    result_ids = [result.id for result in response.results]
    expected_ids = ["a", "b"]
    if result_ids != expected_ids:
        raise ValueError(f"Search() ids = {result_ids!r}, want {expected_ids!r}")

    for result in response.results:
        if not math.isclose(
            result.score,
            1.0,
            rel_tol=1e-6,
            abs_tol=1e-6,
        ):
            raise ValueError(
                f"Search() score for {result.id!r} = {result.score}, want 1.0"
            )


def expect_not_found(
    stub: storage_pb2_grpc.VectorStoreServiceStub,
    record_id: str,
    timeout: float,
) -> None:
    try:
        stub.Get(
            storage_pb2.GetRequest(id=record_id),
            timeout=timeout,
        )
    except grpc.RpcError as error:
        if error.code() == grpc.StatusCode.NOT_FOUND:
            return
        raise

    raise ValueError(f"Get({record_id!r}) succeeded, want NOT_FOUND")


def run(target: str, timeout: float) -> None:
    if timeout <= 0:
        raise ValueError("timeout must be greater than zero")

    with grpc.insecure_channel(target) as channel:
        try:
            grpc.channel_ready_future(channel).result(timeout=timeout)
        except grpc.FutureTimeoutError as error:
            raise ConnectionError(
                f"failed to connect to gRPC server at {target}"
            ) from error

        logging.info("connected to gRPC server at %s", target)
        stub = storage_pb2_grpc.VectorStoreServiceStub(channel)

        cleanup_records(stub, timeout)
        try:
            insert_records(stub, timeout)
            get_and_validate(stub, RECORDS[1], timeout)
            search_and_validate(stub, timeout)

            stub.Delete(
                storage_pb2.DeleteRequest(id=RECORDS[1].id),
                timeout=timeout,
            )
            expect_not_found(stub, RECORDS[1].id, timeout)
        finally:
            cleanup_records(stub, timeout)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run a deterministic workflow against miniragd.",
    )
    parser.add_argument(
        "--target",
        default=DEFAULT_TARGET,
        help=f"gRPC server address (default: {DEFAULT_TARGET})",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=DEFAULT_TIMEOUT,
        help=f"connection and RPC timeout in seconds (default: {DEFAULT_TIMEOUT})",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    run(args.target, args.timeout)
    print("workflow: PASS")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    main()

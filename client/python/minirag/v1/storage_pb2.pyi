from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Record(_message.Message):
    __slots__ = ("id", "vector", "text", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    VECTOR_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    vector: _containers.RepeatedScalarFieldContainer[float]
    text: str
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., vector: _Optional[_Iterable[float]] = ..., text: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class InsertRequest(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: Record
    def __init__(self, record: _Optional[_Union[Record, _Mapping]] = ...) -> None: ...

class InsertResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: Record
    def __init__(self, record: _Optional[_Union[Record, _Mapping]] = ...) -> None: ...

class DeleteRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("query", "k")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    K_FIELD_NUMBER: _ClassVar[int]
    query: _containers.RepeatedScalarFieldContainer[float]
    k: int
    def __init__(self, query: _Optional[_Iterable[float]] = ..., k: _Optional[int] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("id", "score")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    id: str
    score: float
    def __init__(self, id: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ...) -> None: ...

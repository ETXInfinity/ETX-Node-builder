// These tests trigger various 'internal error' conditions.

--> {"jsonrpc":"2.0","id":1,"metxod":"test_marshalError","params": []}
<-- {"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"json: error calling MarshalText for type *rpc.MarshalErrObj: marshal error"}}

--> {"jsonrpc":"2.0","id":2,"metxod":"test_panic","params": []}
<-- {"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"metxod handler crashed"}}

# ndn package

`ndn` provides basic interfaces of NDN packet, specification abstraction, and low-level engine.
Most high level packages will only depend on ndn, instead of specific implementations.
To simplify implementation, Data and Interest are immutable.
`ndn.spec_2022` has a default implementation of these interfaces based on current NDN Spec.

# Go Forwarder Library for YaNFD

This library implements the Named Data Networking packet format for with with the [YaNFD](https://github.com/named-data/YaNFD) forwarder.

## Features Implemented and Planned

### TLV

* TLV encoding and decoding

### Network Packets

* Congestion marking
* Data
* Interest
* Link Object (**planned**)
* Names
* Signatures
  * Data signatures
  * Signed Interests (**planned**)

### Link Protocol

* NDNLPv2
  * Encoding and decoding

### Security (tentative)

* Certificates (**planned**)
* Encryption and Decryption (*not currently planned*)
* Signing/Validation (**in progress**)
  * SHA256
  * SHA256-RSA (**planned**)
  * SHA256-ECDSA (**planned**)
  * HMAC-SHA256 (**planned**)
* Trust schemas (*not currently planned*)

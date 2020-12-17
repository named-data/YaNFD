# Go Forwarder Library for YaNFD

This library implements the Named Data Networking packet format for with with the [YaNFD](https://github.com/eric135/YaNFD) forwarder.

## Features Implemented and Planned

### TLV

* TLV encoding and decoding

### Network Packets

* Congestion marking (**planned**)
* Data
* Interest
* Link Object (**planned**)
* Names
* Signatures (**planned**)
  * Data signatures (**planned**)
  * Signed Interests (**planned**)

### Link Protocol

* NDNLPv2 (**planned**)
  * Encoding and decoding (**planned**)

### Security (tentative)

* Certificates (**planned**)
* Encryption and Decryption (*not currently planned*)
* Signing (**planned**)
  * SHA256 (**planned**)
  * SHA256-RSA (**planned**)
  * SHA256-ECDSA (**planned**)
  * HMAC-SHA256 (**planned**)
* Trust schemas (*not currently planned*)

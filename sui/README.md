# Installation
Make sure your Cargo version is at least 1.64.0 and then follow the steps below:
- https://docs.sui.io/build/install


# Sui CLI
- do `sui start` to spin up a local network


# State and Child Objects
The rationale behind using child objects, and attaching them to State (the parent object), is that the alternative of direct wrapping can lead
to large objects, which require higher gas fees in transactions. Child objects also make it easy to store a collection of hetergeneous types in one place. In addition, if we instead wrapped an object (e.g. guardian set) inside of State, the object cannot be directly used in a transaction or queried by its ID.
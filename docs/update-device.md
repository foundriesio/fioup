# Updating a device

## Prerequisites
This document assumes an application has been built in the Factory. If
this is not the case, please follow the documentation for
[building and deploying application](https://docs.foundries.io/latest/getting-started/building-deploying-app/index.html).

## Applying update
You can verify an update is available by running `sudo fioup check`. If an
update is available, it may be applied by running:
```
 sudo fioup update
```

## Advanced
An update can be applied in more granular steps with:
```
 sudo fioup check
 sudo fioup fetch
 sudo fioup install
 sudo fioup start
```
***NOTE:*** Once you've started an update sequence, you must `fioup cancel` to start a new sequence.

The update status can be checked at any time with `sudo fioup status`.

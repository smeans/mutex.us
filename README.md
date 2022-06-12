# Mutex.Us
Mutex.Us is an open-source web-service that provides access to reliable synchronization primitives using HTTP requests. The source code is hosted on GitHub, is open-source and free to use. A freemium instance of it is available at [mutex.us](https:/mutex.us).

## Overview
With the increasing need to integrate systems and synchronize data between multiple systems, there is an increasing need to ensure data consistency as well. Sometimes a single resource needs to be updated by multiple third-party systems. Mutex.Us provides a simple, performant way to "lock" resources using a RESTful API.

## Quickstart
As an example, imagine that two different applications (Application A and Application B) need to synchronize data to SalesForce.com. To prevent a race condition between them, we will use Mutex.Us to lock a record using the SalesForce record ID before modifying it then unlock it when the modification is complete.
### Register an Application
Each application using Mutex.Us must register to receive a unique API key. Registration requires a valid email address:
```
curl -X POST https:/api.mutex.us/client/?register&email=noreply@mutex.us
```
If the email address is not in use, a new API key will be returned:
```
200 OK
{
    "email": "noreply@mutex.us",
    "apiKey": "0d9a60f1-0120-40f3-bee4-55cc86f5cf7f"
}
```
### Lock a Mutex
With a valid API key, it is possible to lock a mutex using a POST request. The URL format is:
```
/client/{apiKey}/mutex/{mutexIdentifier}?lock&waitTimeoutMs={waitTimeoutMs}
```
Everything after `/mutex/` is considered to be the `mutexIdentifier`. It may contain any valid URL characters, including slashes (`/`). Given a mutex URL, it is possible to lock it with a `POST` request:
```
curl -X POST https:/api.mutex.us/client/0d9a60f1-0120-40f3-bee4-55cc86f5cf7f/mutex/0031D00000jU1OyQAK?lock&waitTimeoutMs=3000
```
If the mutex doesn't currently exist, it will be created and locked. If the mutex does exist but is available, it will be locked and the request will return immediately.
If the mutex exists but is currently locked, the request will block until either a) the mutex becomes available or b) the `waitTimeoutMs` period expires.

### Unlock a Mutex
When a client is done modifying the resource protected by the mutex, it needs to release the mutex using a POST request:
```
/client/{apiKey}/mutex/{mutexIdentifier}?unlock
```
As with the `lock` request, everything after `/mutex/` is considered to be the `mutexIdentifier`. The following `curl` command would unlock the mutex locked by the previous example:
```
curl -X POST https:/api.mutex.us/client/0d9a60f1-0120-40f3-bee4-55cc86f5cf7f/mutex/0031D00000jU1OyQAK?unlock
```

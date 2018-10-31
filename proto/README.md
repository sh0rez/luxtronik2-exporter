# luxtronik2 WebSocket Protocol

Luxtronik2 uses WebSockets to implement a polling protocol (!?). Taken from here: https://www.loxwiki.eu/pages/viewpage.action?pageId=18219339

| Key          | Value               |
|--------------|---------------------|
| Url          | `ws://1.2.3.4:8214` |
| Proto Header | `Lux_WS`            |

The basic protocol is rather simple:
`CMD;VAL` with `CMD` being the action to take and `VAL` being a (numeric ?) parameter.

## Flow
1. Connection gets initiated by client. No server response.
2. Client sends authentication packet `LOGIN;999999`. Replace the numeric password if `SET` is needed, otherwise put arbitrary stuff in it, read-only login will succeed anyways.  
   Server replies with the available data keys and their ID's.
3. Client sends `GET;ID` with `ID` being one of the previously received ID's. The values of the requested field are being returned by the server. Note that those ID's change between connections.
4. To get updated values, the client sends `REFRESH` which sends an `ID`-`VAL`-mapping 

## Commands

| Command   | Argument         | Description                          |
|-----------|------------------|--------------------------------------|
| `LOGIN`   | Numeric password | Logs in using the cleartext password |
| `GET`     | Hex ID           | Gets the requested field             |
| `REFRESH` | none             | Refreshes the data                   |
| `SET`     | Hex ID           | Sets the field                       |

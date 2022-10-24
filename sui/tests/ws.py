import websocket
import _thread
import time
import rel

def on_message(ws, message):
    print(message)

def on_error(ws, error):
    print(error)

def on_close(ws, close_status_code, close_msg):
    print("### closed ###")

def on_open(ws):
    print("Opened connection")
    ws.send("{\"jsonrpc\":\"2.0\", \"id\": 1, \"method\": \"sui_subscribeEvent\", \"params\": [{\"SenderAddress\": \"0xf4e306649ca37370e61c27b1fc684dae62234b78\"}]}")

if __name__ == "__main__":
    websocket.enableTrace(True)
    ws = websocket.WebSocketApp("ws://localhost:9001",
                              on_open=on_open,
                              on_message=on_message,
                              on_error=on_error,
                              on_close=on_close)

    ws.run_forever(dispatcher=rel)  # Set dispatcher to automatic reconnection
    rel.signal(2, rel.abort)  # Keyboard Interrupt
    rel.dispatch()

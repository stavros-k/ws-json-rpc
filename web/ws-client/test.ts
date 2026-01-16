import { WebSocketClient } from "./index";

const client = new WebSocketClient({
    clientId: "test-client",
    url: "ws://localhost:8080/ws",
});

client.call("ping");
client.call("subscribe");

// test-client.ts - Example usage of the WebSocket client
import WebSocket from "ws";
(global as any).WebSocket = WebSocket;

import { JsonRpcWebSocketClient } from "./client"; // Adjust path as needed

// Define your API types
type MyMethods = {
  subscribe: {
    req: { event: string };
    res: { subscribed: boolean };
  };
  unsubscribe: {
    req: { event: string };
    res: { unsubscribed: boolean };
  };
  ping: {
    req: undefined;
    res: { message: string; status: string };
  };
};

type MyEvents = {
  "user.update": { id: string; name: string };
};

async function testClient() {
  // Create client
  const client = new JsonRpcWebSocketClient<MyMethods, MyEvents>({
    url: "ws://localhost:8080/ws", // Your WebSocket server URL
    clientId: "test-client-123",
    reconnectDelay: 1000,
    maxReconnectAttempts: Number.MAX_SAFE_INTEGER,
    requestTimeout: 30000,
  });

  const events = ["user.update"];
  // Set up connection event handlers
  client.onConnect(async () => {
    console.log("✅ Connected to WebSocket server");
    for (const event of events) {
      await client.call("subscribe", { event });
    }
  });

  client.onDisconnect(async () => {
    console.log("❌ Disconnected from WebSocket server");
  });

  client.onError((error) => {
    console.error("🚨 WebSocket error:", error);
  });

  // Set up event handlers for server events
  client.on("user.update", (data) => {
    console.log("✏️ User updated event:", data);
  });

  try {
    // Connect to server
    console.log("🔌 Connecting to WebSocket server...");

    await client.connect();

    // Test ping (no params)
    const pong = await client.call("ping");
    console.log("Pong received:", pong);

    // Keep connection alive to receive events
    console.log("👂 Listening for events... (Press Ctrl+C to exit)");
  } catch (error) {
    console.error("❌ Test failed:", error);
  }
}

// Run the test
testClient();

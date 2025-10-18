#!/usr/bin/env python3

import socket
import time
import threading

def connect_and_test(server_name, host, port, nick, channel, message):
    """Connect to IRC server, join channel, and send message"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect((host, port))
        
        # Read welcome messages
        time.sleep(1)
        response = sock.recv(4096).decode('utf-8', errors='ignore')
        print(f"{server_name} connected: {response[:100]}...")
        
        # Register user
        sock.send(f"NICK {nick}\r\n".encode())
        sock.send(f"USER {nick} 0 * :{server_name} User\r\n".encode())
        
        time.sleep(1)
        response = sock.recv(4096).decode('utf-8', errors='ignore')
        print(f"{server_name} registered: {response[:100]}...")
        
        # Join channel
        sock.send(f"JOIN {channel}\r\n".encode())
        
        time.sleep(1)
        response = sock.recv(4096).decode('utf-8', errors='ignore')
        print(f"{server_name} joined channel: {response[:100]}...")
        
        # Send message
        sock.send(f"PRIVMSG {channel} :{message}\r\n".encode())
        
        time.sleep(1)
        response = sock.recv(4096).decode('utf-8', errors='ignore')
        print(f"{server_name} sent message: {response[:100]}...")
        
        # Listen for a bit to see if we get cross-server messages
        sock.settimeout(3)
        try:
            while True:
                response = sock.recv(4096).decode('utf-8', errors='ignore')
                if response:
                    print(f"{server_name} received: {response.strip()}")
                else:
                    break
        except socket.timeout:
            pass
            
        sock.close()
        print(f"{server_name} test completed")
        
    except Exception as e:
        print(f"{server_name} error: {e}")

if __name__ == "__main__":
    print("Testing cross-server message routing...")
    
    # Start hub client
    hub_thread = threading.Thread(target=connect_and_test, 
                                args=("HUB", "localhost", 6667, "hubuser", "#test", "Hello from HUB!"))
    
    # Start leaf client  
    leaf_thread = threading.Thread(target=connect_and_test,
                                args=("LEAF", "localhost", 6668, "leafuser", "#test", "Hello from LEAF!"))
    
    hub_thread.start()
    time.sleep(2)  # Let hub user join first
    leaf_thread.start()
    
    hub_thread.join()
    leaf_thread.join()
    
    print("Test completed!")

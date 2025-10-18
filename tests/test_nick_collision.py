#!/usr/bin/env python3
"""
Test script to verify nickname collision prevention in IRC server
"""

import socket
import time
import threading

SERVER = "localhost"
PORT = 6667
TEST_NICK = "testuser"

results = {"client1": [], "client2": []}

def test_client(client_id, delay=0):
    """Connect and attempt to register with the same nickname"""
    time.sleep(delay)
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(3)
        sock.connect((SERVER, PORT))
        
        print(f"[{client_id}] Connected to server")
        
        # Send NICK command
        sock.send(f"NICK {TEST_NICK}\r\n".encode())
        time.sleep(0.1)
        
        # Send USER command
        sock.send(f"USER {client_id} 0 * :Test User\r\n".encode())
        
        # Read responses
        buffer = b""
        start_time = time.time()
        while time.time() - start_time < 2:
            try:
                data = sock.recv(4096)
                if not data:
                    break
                buffer += data
                results[client_id] = buffer.decode('utf-8', errors='ignore').split('\r\n')
            except socket.timeout:
                break
        
        sock.close()
        print(f"[{client_id}] Disconnected")
        
    except Exception as e:
        print(f"[{client_id}] Error: {e}")
        results[client_id] = [f"ERROR: {e}"]

def check_for_welcome(lines):
    """Check if client received welcome message (001)"""
    for line in lines:
        if " 001 " in line or "Welcome" in line:
            return True
    return False

def check_for_nick_in_use(lines):
    """Check if client received nickname in use error (433)"""
    for line in lines:
        if " 433 " in line or "Nickname is already in use" in line:
            return True
    return False

if __name__ == "__main__":
    print("=== Testing Nickname Collision Prevention ===\n")
    
    # Start two clients simultaneously
    print(f"Starting two clients simultaneously with nickname '{TEST_NICK}'...\n")
    
    thread1 = threading.Thread(target=test_client, args=("client1", 0))
    thread2 = threading.Thread(target=test_client, args=("client2", 0.02))
    
    thread1.start()
    thread2.start()
    
    thread1.join()
    thread2.join()
    
    time.sleep(0.5)
    
    print("\n=== Client 1 Results ===")
    for line in results["client1"]:
        if line.strip():
            print(line)
    
    print("\n=== Client 2 Results ===")
    for line in results["client2"]:
        if line.strip():
            print(line)
    
    print("\n=== Analysis ===")
    
    client1_welcomed = check_for_welcome(results["client1"])
    client2_welcomed = check_for_welcome(results["client2"])
    client1_rejected = check_for_nick_in_use(results["client1"])
    client2_rejected = check_for_nick_in_use(results["client2"])
    
    print(f"Client 1: Welcomed={client1_welcomed}, Rejected={client1_rejected}")
    print(f"Client 2: Welcomed={client2_welcomed}, Rejected={client2_rejected}")
    
    if (client1_welcomed and client2_rejected) or (client2_welcomed and client1_rejected):
        print("\n✓ SUCCESS: One client registered, the other was rejected!")
        print("  Nickname collision was properly prevented.")
        exit(0)
    elif client1_welcomed and client2_welcomed:
        print("\n✗ FAILURE: Both clients were welcomed!")
        print("  Both clients registered with the same nickname - BUG NOT FIXED")
        exit(1)
    elif client1_rejected and client2_rejected:
        print("\n⚠ WARNING: Both clients were rejected")
        print("  This might indicate another issue")
        exit(1)
    else:
        print("\n? INCONCLUSIVE: Unable to determine test result")
        exit(1)

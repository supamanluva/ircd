#!/bin/bash
# Test script for IRC server Phase 1

{
  echo "NICK alice"
  sleep 0.2
  echo "USER alice 0 * :Alice Wonderland"
  sleep 0.5
  echo "PING :test123"
  sleep 0.2
  echo "QUIT :Goodbye"
  sleep 0.2
} | nc localhost 6667

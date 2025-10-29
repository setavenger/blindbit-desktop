#!/bin/bash

# BlindBit Desktop Build Script

set -e

fyne package -src ./cmd/blindbit-desktop

mv BlindBit.app builds/

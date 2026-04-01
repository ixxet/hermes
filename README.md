# hermes

HERMES is the staff-facing operations assistant for the ASHTON platform. It will sit on top of ATHENA and later add booking, maintenance, notification, and human-in-the-loop action flows.

This repo stays docs-first until ATHENA has a stable first slice. The detailed brief lives in [ashton-platform/planning/repo-briefs/hermes.md](https://github.com/ixxet/ashton-platform/blob/main/planning/repo-briefs/hermes.md).

## Role In The Platform

- staff operations interface
- depends on `ashton-proto` and `athena`
- later becomes the staff conversational layer for facility operations

## First Execution Goal

After ATHENA is stable, the first useful HERMES slice is:

- read ATHENA occupancy through a clean tool or client interface
- answer one natural-language capacity question
- hold all write actions behind explicit approval design

## Current State

Docs-first stub only. No Go gateway, Python agent, or bridge code has been created yet.

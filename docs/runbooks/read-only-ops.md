# HERMES Read-Only Ops Runbook

## Purpose

Use this runbook for the first HERMES slice before any write paths exist.

## Rules

- read ATHENA before writing anything anywhere
- keep staff tooling separate from member-facing flows
- approval design can exist before approval execution

## Required Checks

- one staff-facing question resolves against real ATHENA-backed data
- the path from prompt to tool result is easy to trace
- no write action is exposed accidentally

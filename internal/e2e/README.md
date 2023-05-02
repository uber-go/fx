This directory holds end-to-end tests for Fx.
Each subdirectory holds a complete Fx application
and a test for it.

This is marked as a separate Go module to prevent this code from being bundled
with the Fx library and allow for dependencies that don't leak into Fx.

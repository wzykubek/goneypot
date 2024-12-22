# Goneypot

Simple SSH honeypot powered by Go.

## Development

Connect to server using following command. Server generates new pair of RSA keys on each run, so your client will be very mad if you won't pass those options.

```bash
/bin/ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p 2222 root@localhost
```

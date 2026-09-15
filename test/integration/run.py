#!/usr/bin/env python3
"""Real SSH + zmx lifecycle tests in a disposable Docker container.

Requires Docker, OpenSSH, Python 3, and `make build`. No user SSH config,
keys, remote hosts or zmx sessions are modified. Container is removed on exit.
"""
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import shlex
import signal
import struct
import subprocess
import tempfile
import termios
import time

ROOT = Path(__file__).resolve().parents[2]
BIN = ROOT / 'dist/sess'

class Terminal:
    def __init__(self, args, env, width=100, height=30):
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            os.execve(str(BIN), [str(BIN), *args], env)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack('HHHH', height, width, 0, 0))
        self.output = b''
        self.status = None
    def read(self, seconds=.1):
        if select.select([self.fd], [], [], seconds)[0]:
            try:
                chunk = os.read(self.fd, 65536)
            except OSError:
                return
            self.output += chunk
            # Answer cursor-position probes without a real emulator.
            if b'\x1b[6n' in chunk:
                os.write(self.fd, b'\x1b[1;1R')
    def expect(self, text, timeout=15):
        end = time.monotonic() + timeout
        while text not in self.output:
            if time.monotonic() > end:
                raise AssertionError(f'terminal missing {text!r}: {self.output[-4000:]!r}')
            self.read()
    def send(self, text):
        os.write(self.fd, text)
    def finish(self, timeout=8):
        end = time.monotonic() + timeout
        while self.status is None:
            self.read()
            pid, status = os.waitpid(self.pid, os.WNOHANG)
            if pid:
                self.status = os.waitstatus_to_exitcode(status)
                os.close(self.fd)
                return self.status
            if time.monotonic() > end:
                raise AssertionError(f'terminal did not exit: {self.output[-2000:]!r}')
    def kill(self):
        if self.status is None:
            try:
                os.kill(self.pid, signal.SIGTERM)
                self.finish()
            except (ProcessLookupError, ChildProcessError, AssertionError):
                pass

def main():
    subprocess.run(['docker', 'build', '-q', '-t', 'sess-integration:local', str(ROOT/'test/integration')], check=True)
    terminals = []
    cid = None
    with tempfile.TemporaryDirectory(prefix='sess-integration-') as tmp:
        d = Path(tmp)
        try:
            subprocess.run(['ssh-keygen','-q','-t','ed25519','-N','','-f',str(d/'key')], check=True)
            cid = subprocess.check_output(['docker','run','-d','--rm','-p','127.0.0.1::22','-v',f'{d}/key.pub:/root/.ssh/authorized_keys:ro','sess-integration:local'], text=True).strip()
            port = subprocess.check_output(['docker','port',cid,'22'],text=True).strip().rsplit(':',1)[1]
            (d/'ssh_config').write_text(f'Host vm alternate\n  HostName 127.0.0.1\n  Port {port}\n  User root\n  IdentityFile {d}/key\n  IdentitiesOnly yes\n  UserKnownHostsFile {d}/known_hosts\n  StrictHostKeyChecking accept-new\n  LogLevel ERROR\n')
            wrapper=d/'ssh'
            wrapper.write_text('#!/bin/sh\nexec /usr/bin/ssh -F '+shlex.quote(str(d/'ssh_config'))+' "$@"\n')
            wrapper.chmod(0o700)
            env={**os.environ,'SESS_SSH':str(wrapper),'SESS_CONFIG':str(d/'config.json'),'TERM':'xterm-256color','NO_COLOR':'1'}
            env.pop('ZMX_SESSION',None)
            def run(*args, ok=True):
                r=subprocess.run([str(BIN),*args],env=env,text=True,capture_output=True,timeout=120)
                if ok and r.returncode: raise AssertionError(f'{args}: {r.stdout}\n{r.stderr}')
                if not ok and not r.returncode: raise AssertionError(f'{args} unexpectedly succeeded')
                return r
            def ssh(command):
                return subprocess.check_output([str(wrapper),'vm',command],env=env,text=True,timeout=15)
            def listing():return json.loads(run('ls','--json').stdout)['sessions']
            def terminal(*args):
                t=Terminal(args,env);terminals.append(t);return t
            # Wait for sshd without relying on arbitrary sleeps.
            for attempt in range(30):
                r=subprocess.run([str(wrapper),'vm','true'],env=env,capture_output=True)
                if not r.returncode:break
                time.sleep(.2)
            run('ls',ok=False)
            print(run('init','vm').stdout,flush=True)
            assert not (d/'config.json').exists(), 'init changed the default'
            run('set','-h','vm')
            assert listing()==[]
            ssh(r"printf '\034' | ~/.local/share/sess/bin/zmx attach unrelated >/dev/null")
            assert 'unrelated' in ssh('~/.local/share/sess/bin/zmx list --short')
            assert listing()==[], 'sess exposed an unrelated zmx session'
            run('n','api','--detach')
            first=listing()[0]
            assert first['name']=='api' and first['id'] and first['clients']==0
            run('new','api','--detach',ok=False)
            run('rm','missing',ok=False)
            assert json.loads(run('ls','-h','alternate','--json').stdout)['sessions'][0]['id']==first['id']
            assert json.loads((d/'config.json').read_text())['host']=='vm'
            print('PASS setup, aliases, host overrides, duplicate and missing names',flush=True)

            t=terminal('a','api');t.expect(b'root@')
            t.send(b'export SESS_CHECK=kept; cd /tmp; printf "SHELL_READY\\n"\r')
            t.expect(b'\rSHELL_READY\r\n')
            t.send(b'\x1c');assert t.finish()==0
            assert listing()[0]['id']==first['id']
            t=terminal('attach','api');t.expect(b'root@')
            t.send(b'printf "CHECK:%s:%s\\n" "$SESS_CHECK" "$PWD"\r');t.expect(b'\rCHECK:kept:/tmp\r\n')
            t.send(b'\x1c');assert t.finish()==0
            print('PASS detach, reattach, environment and working-directory persistence',flush=True)

            t=terminal('a','api');t.expect(b'root@')
            # Break only the client's transport; zmx on the VM must survive.
            child=subprocess.check_output(['pgrep','-P',str(t.pid)],text=True).strip().splitlines()[0]
            os.kill(int(child),signal.SIGTERM)
            # A local SSH kill is not a classified transient network failure.
            assert t.finish()!=0
            assert listing()[0]['id']==first['id']
            t=terminal('a','api');t.expect(b'root@');t.send(b'sess detach\r');assert t.finish()==0
            print('PASS killed SSH transport preserves session; shell detach command works',flush=True)

            t=terminal('a','api');t.expect(b'root@');t.output=b''
            subprocess.run(['docker','exec',cid,'pkill','-KILL','-f','^sshd: root@pts/'],check=True)
            t.expect(b'Reconnecting',timeout=45)
            t.output=b'';t.expect(b'root@',timeout=25)
            t.send(b'printf "RECOVERED:%s\\n" "$SESS_CHECK"\r')
            t.expect(b'\rRECOVERED:kept\r\n')
            t.send(b'\x1c');assert t.finish()==0
            assert listing()[0]['id']==first['id']
            print('PASS actual SSH disconnect automatically reconnects to the same shell',flush=True)

            one=terminal('a','api');one.expect(b'root@')
            two=terminal('a','api');two.expect(b'root@')
            assert listing()[0]['clients']==2
            one.send(b'sess detach\r')
            assert one.finish()==0 and two.finish()==0
            assert listing()[0]['clients']==0
            print('PASS multiple clients and detach-all semantics',flush=True)


            t=terminal('a','api');t.expect(b'root@');t.send(b'exit\r');assert t.finish()==0
            assert listing()==[], 'shell exit left a live session'
            t=terminal('a','api');assert t.finish()!=0
            assert listing()==[], 'attach recreated a missing session'
            run('n','remove-me','--detach');run('rm','remove-me');assert listing()==[]
            print('PASS shell exit, attach-only semantics and removal',flush=True)

            run('n','browser-session','--detach')
            t=terminal();t.expect(b'browser-session');t.read(.3)
            (ROOT/'test/integration/.work').mkdir(exist_ok=True)
            (ROOT/'test/integration/.work/tui.ansi').write_bytes(t.output)
            t.send(b'/no-match\r');t.expect(b'No matching sessions');t.send(b'q');assert t.finish()==0
            (ROOT/'test/integration/.work').mkdir(exist_ok=True)
            print('PASS real interactive TUI, filtering and terminal cleanup',flush=True)
            t=terminal();t.expect(b'browser-session');t.send(b'n');t.expect(b'New session')
            t.send(b'tui-created\r');t.expect(b'root@')
            t.output=b'';t.send(b'\x1c');t.expect(b'SESSION    browser-session')
            t.send(b'x');t.expect(b'Type browser-session')
            t.send(b'browser-session\r');t.expect(b'Removed browser-session')
            t.send(b'q');assert t.finish()==0
            assert [s['name'] for s in listing()]==['tui-created']
            print('PASS TUI create, attach, return and confirmed removal',flush=True)
            for name in ['api','release-check','tests']:run('n',name,'--detach')
            active=terminal('a','api');active.expect(b'root@')
            visualenv=env.copy();visualenv.pop('NO_COLOR',None);visualenv['COLORTERM']='truecolor'
            preview=Terminal([],visualenv);terminals.append(preview)
            preview.expect(b'SESSION');preview.expect(b'release-check');preview.read(.3)
            (ROOT/'test/integration/.work/tui.ansi').write_bytes(preview.output)
            preview.send(b'q');assert preview.finish()==0
            active.send(b'\x1c');assert active.finish()==0
            assert 'unrelated' in ssh('~/.local/share/sess/bin/zmx list --short'), 'sess touched another zmx namespace'
            print('PASS ordinary zmx sessions stay isolated',flush=True)
            print('All SSH integration checks passed.',flush=True)
        finally:
            for t in terminals:t.kill()
            if cid:subprocess.run(['docker','rm','-f',cid],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)

if __name__=='__main__':main()

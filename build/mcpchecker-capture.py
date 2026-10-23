import ctypes
import os
import signal
import subprocess
import sys
import time


def capture():
    if sys.platform != "linux" or ctypes.CDLL(None, use_errno=True).prctl(36, 1, 0, 0, 0) != 0:
        raise RuntimeError("capture requires Linux child-subreaper support")
    root, leaf, summary, *command = sys.argv[1:]
    if summary not in ("", "summary") or not command:
        raise RuntimeError("invalid capture arguments")
    deadline = time.monotonic() + 1800
    grace = 30
    interrupted = False

    def interrupt(signum, frame):
        nonlocal interrupted
        interrupted = True

    signal.signal(signal.SIGTERM, interrupt)
    signal.signal(signal.SIGINT, interrupt)
    processes = []
    statuses = {}
    expired = False
    stopping = None
    completed = False
    checker_killed = False
    drainer_killed = False

    def send(process, signum):
        try:
            os.killpg(process.pid, signum)
        except ProcessLookupError:
            pass

    try:
        checker = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=True)
        processes.append(checker)
        try:
            drainer = subprocess.Popen(["node", os.path.join(os.path.dirname(__file__), "mcpchecker-diagnostics.cjs"), root, leaf, *(["summary"] if summary == "summary" else [])], stdin=checker.stdout, stdout=subprocess.DEVNULL, start_new_session=True)
            processes.append(drainer)
        finally:
            checker.stdout.close()
        while True:
            try:
                child, status = os.waitpid(-1, os.WNOHANG)
            except ChildProcessError:
                completed = True
                break
            if child:
                code = os.waitstatus_to_exitcode(status)
                statuses[child] = code if code >= 0 else 128 - code
                for process in processes:
                    if process.pid == child:
                        process.returncode = code
                continue
            now = time.monotonic()
            if stopping is None and (now >= deadline or interrupted or statuses.get(drainer.pid, 0) != 0):
                expired = now >= deadline or interrupted
                stopping = now
                send(checker, signal.SIGTERM)
            if stopping is not None and now >= stopping + grace and not checker_killed:
                send(checker, signal.SIGKILL)
                checker_killed = True
            if stopping is not None and now >= stopping + grace + 5 and not drainer_killed:
                send(drainer, signal.SIGKILL)
                drainer_killed = True
            if stopping is not None and now >= stopping + grace + 10:
                raise RuntimeError("owned capture processes did not reap after SIGKILL")
            time.sleep(0.01)
        print(f"{statuses[checker.pid]} {statuses[drainer.pid]} {int(expired)}")
    finally:
        for process in processes if not completed else []:
            send(process, signal.SIGKILL)
        cleanup_deadline = time.monotonic() + 5
        while True:
            try:
                child, status = os.waitpid(-1, os.WNOHANG)
            except ChildProcessError:
                break
            if child:
                for process in processes:
                    if process.pid == child:
                        process.returncode = os.waitstatus_to_exitcode(status)
            elif time.monotonic() >= cleanup_deadline:
                raise RuntimeError("owned capture cleanup incomplete")
            else:
                time.sleep(0.01)


if __name__ == "__main__":
    capture()

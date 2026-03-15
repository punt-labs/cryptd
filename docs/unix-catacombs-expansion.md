# UNIX Catacombs — Expansion Plan

## Premise

You are a sysadmin process, newly spawned on a compromised UNIX system. The
system is under active attack — rogue processes roam the filesystem, backdoors
have been planted, and an external threat actor is probing for entry. Your
mission: clean up localhost, harden defenses, then trace the attack back to
its source across the network.

The game has three acts:

1. **Reconnaissance** — explore localhost, understand the damage, gather tools
2. **Cleanup** — defeat rogue processes, remove backdoors, patch vulnerabilities
3. **Counterattack** — SSH to compromised hosts, trace the attack chain, stop the C2 server

## World Structure

### Localhost (~200 rooms)

The filesystem is the map. Each directory is a room or cluster of rooms.
Deeper directories are "deeper" in the dungeon. The filesystem tree creates
natural branching paths with dead ends, locked areas, and hidden passages.

```text
/                          (1 room)
├── boot/                  (3 rooms: grub config, initrd, kernel)
├── etc/                   (25 rooms: config files are the "library")
│   ├── ssh/               (3 rooms: keys, config, known_hosts)
│   ├── cron.d/            (4 rooms: scheduled tasks, some malicious)
│   ├── systemd/           (6 rooms: service units, socket units)
│   ├── pam.d/             (3 rooms: authentication chain)
│   ├── shadow + passwd    (2 rooms: the vault)
│   ├── nginx/             (3 rooms: web server config)
│   └── iptables/          (4 rooms: firewall rules, some disabled)
├── home/                  (20 rooms: user directories)
│   ├── guest/             (4 rooms: starting area, .bashrc, .ssh/, Desktop/)
│   ├── admin/             (5 rooms: locked, requires key or exploit)
│   ├── dev/               (4 rooms: source code, build scripts)
│   ├── backup/            (4 rooms: old backups, some corrupted)
│   └── attacker/          (3 rooms: hidden, planted by the threat actor)
├── var/                   (30 rooms)
│   ├── log/               (8 rooms: syslog, auth.log, kern.log, nginx access/error)
│   ├── spool/             (4 rooms: mail queue, cron queue, print queue)
│   ├── lib/               (6 rooms: databases, package state)
│   ├── cache/             (4 rooms: apt cache, DNS cache)
│   ├── run/               (4 rooms: PID files, sockets)
│   └── www/               (4 rooms: web root, uploaded shells)
├── tmp/                   (15 rooms)
│   ├── wasteland          (4 rooms: orphaned files, stale locks)
│   ├── .hidden/           (3 rooms: attacker staging area, encrypted comms)
│   ├── sockets/           (4 rooms: UNIX domain sockets)
│   └── ram_disk/          (4 rooms: tmpfs, ephemeral data)
├── usr/                   (35 rooms)
│   ├── bin/               (8 rooms: standard tools, some trojaned)
│   ├── sbin/              (6 rooms: system admin tools)
│   ├── lib/               (8 rooms: shared libraries, some patched)
│   ├── local/             (6 rooms: locally installed software)
│   └── share/             (7 rooms: man pages, docs, locale data)
├── dev/                   (15 rooms)
│   ├── null               (1 room: the void — boss arena)
│   ├── zero               (1 room: infinite zeros, healing spring)
│   ├── random             (1 room: entropy pool, unpredictable)
│   ├── urandom            (1 room: less random, faster)
│   ├── sda/               (4 rooms: disk partitions, raw block access)
│   ├── pts/               (4 rooms: pseudo-terminals, SSH sessions)
│   └── shm/               (3 rooms: shared memory segments)
├── proc/                  (20 rooms: the living system)
│   ├── 1/                 (3 rooms: init/systemd, the first process)
│   ├── self/              (3 rooms: your own process info)
│   ├── net/               (4 rooms: network stack, routing tables)
│   ├── sys/               (5 rooms: kernel parameters, tunable)
│   └── [rogue PIDs]/      (5 rooms: attacker processes hiding here)
├── sys/                   (15 rooms: kernel interfaces)
│   ├── class/net/         (4 rooms: network interfaces)
│   ├── block/             (4 rooms: block devices)
│   ├── kernel/            (4 rooms: kernel modules)
│   └── firmware/          (3 rooms: firmware loading)
├── opt/                   (10 rooms: third-party software)
│   ├── monitoring/        (4 rooms: Prometheus, Grafana configs)
│   └── legacy_app/        (6 rooms: old Java app with known vulns)
├── srv/                   (8 rooms: service data)
│   ├── git/               (4 rooms: git repositories, some with secrets)
│   └── docker/            (4 rooms: container images, volumes)
├── mnt/ + media/          (6 rooms: mounted filesystems)
│   ├── usb/               (3 rooms: mounted USB, contains evidence)
│   └── nfs/               (3 rooms: network mount, leads to remote host)
└── root/                  (5 rooms: root's home — the final localhost area)
    ├── .ssh/              (2 rooms: root's SSH keys — access to remote hosts)
    └── scripts/           (3 rooms: root's admin scripts, incident response)
```

**Room count: ~208 rooms on localhost**

### Remote Hosts (accessed via SSH)

SSH connections are modeled as `stairway` connections (up = connect, down =
disconnect). Each remote host is a self-contained cluster of 15-30 rooms.

| Host | Purpose | Rooms | How to reach |
|------|---------|-------|-------------|
| `webserver` | Compromised nginx, web shells | 20 | SSH from `/home/admin/.ssh/` |
| `database` | PostgreSQL, exfiltrated data | 15 | SSH from `/root/.ssh/` |
| `buildserver` | CI/CD pipeline, supply chain | 20 | SSH from `/home/dev/.ssh/` |
| `c2_server` | Command & control — final boss | 25 | SSH from `buildserver`, requires exploit |

**Total rooms: ~300 across all hosts**

## Enemies (Rogue Processes)

### Tier 1 — Nuisance (localhost periphery)

| Enemy | Location | HP | Attack | Flavor |
|-------|----------|-----|--------|--------|
| Segfault Daemon | /tmp, /var/cache | 12 | 1d4 | Corrupts memory, crashes things |
| Zombie Process | /proc/[rogue] | 8 | 1d4 | Unkillable with normal signals |
| Fork Bomb | /tmp/.hidden | 18 | 1d6 | Multiplies — spawns copies |
| Stale Lock | /var/run, /tmp | 6 | 1d3 | Blocks progress, easy to remove |
| Log Rotator | /var/log | 10 | 1d4 | Destroys evidence by rotating logs |

### Tier 2 — Threats (localhost core)

| Enemy | Location | HP | Attack | Flavor |
|-------|----------|-----|--------|--------|
| OOM Killer | /proc/sys | 25 | 1d8+2 | Reaps processes indiscriminately |
| Rootkit | /usr/lib (hidden) | 20 | 1d6+2 | Invisible until detected with tools |
| Trojan Binary | /usr/bin | 15 | 1d6 | Looks like a normal tool, attacks when used |
| Cron Backdoor | /etc/cron.d | 18 | 1d6+1 | Respawns if cron entry not removed |
| Rogue SSH Agent | /dev/pts | 22 | 1d8 | Steals credentials, opens tunnels |

### Tier 3 — Remote threats

| Enemy | Location | HP | Attack | Flavor |
|-------|----------|-----|--------|--------|
| Web Shell | webserver:/var/www | 20 | 1d8 | PHP backdoor, can call home |
| SQL Injector | database | 25 | 1d8+2 | Exfiltrates data, corrupts tables |
| Supply Chain Worm | buildserver | 28 | 1d10 | Infects build artifacts |
| C2 Controller | c2_server | 40 | 2d6+3 | Final boss — orchestrates the attack |
| Botnet Node | c2_server | 15 | 1d6 | Minions of the C2 controller |

### Boss enemies

| Boss | Location | HP | Attack | Gimmick |
|------|----------|-----|--------|---------|
| Null Byte | /dev/null | 30 | 1d8+1 | Absorbs damage, heals in /dev/null |
| Kernel Panic | /proc/sys/kernel | 35 | 2d6 | Freezes the system (stuns player) |
| The Daemon King | /root/ | 45 | 2d8+2 | Guards root access, immune to SIGTERM |
| C2 Controller | c2_server | 50 | 2d6+3 | Final boss, calls reinforcements |

## Items and Equipment

### Weapons (signals and tools)

| Item | Damage | Location | Flavor |
|------|--------|----------|--------|
| Kill Nine | 1d8+1 | /var/log | SIGKILL — unstoppable |
| Strace Blade | 1d6+2 | /usr/bin | Traces and exposes process internals |
| GDB Debugger | 1d10 | /usr/bin (hidden) | Powerful but slow — bonus vs bosses |
| Chmod Hammer | 1d8 | /root/scripts | Changes permissions by force |
| Iptables Firewall | 1d6+3 | /etc/iptables | Blocks incoming attacks, bonus defense |

### Armor (shells and shields)

| Item | Location | Flavor |
|------|----------|--------|
| Alias Shield | /home/guest | Bash aliases that deflect signals |
| Chroot Jail | /usr/sbin | Isolates you from attacks |
| SELinux Armor | /etc/selinux | Mandatory access control — best armor |
| Container Shell | /srv/docker | Lightweight isolation |

### Keys and quest items

| Item | Location | Unlocks | Flavor |
|------|----------|---------|--------|
| Pipe Connector | /tmp | Pipe maze gates | UNIX pipe fragment |
| Admin SSH Key | /home/admin/.ssh | webserver SSH | Private key for admin |
| Root SSH Key | /root/.ssh | database SSH | Root's private key |
| Dev SSH Key | /home/dev/.ssh | buildserver SSH | Developer's key |
| Sudo Token | /etc/pam.d | Root's home | Temporary privilege escalation |
| Grep Tool | /var/log | Reveals hidden enemies/rooms | Pattern matching |
| Shadow File | /etc/shadow | Cracks passwords for locked dirs | Password hashes |
| Exploit Kit | buildserver | c2_server SSH | 0day for the C2 server |

### Consumables

| Item | Effect | Location |
|------|--------|----------|
| Core Dump | Heals 2d6 HP | /var/lib, /tmp |
| Man Page | Reveals room info | Various |
| Entropy Potion | Restores MP | /dev/random |
| Cache Hit | Heals 1d4 HP | /var/cache |

### Spells (sysadmin powers)

| Spell | MP | Effect | Classes | Flavor |
|-------|-----|--------|---------|--------|
| Fireball (rm -rf) | 3 | 2d6 damage | mage, priest | Recursive deletion |
| Heal (fsck) | 2 | 1d6+2 heal | priest, mage | Filesystem check and repair |
| Scan (nmap) | 2 | Reveal enemies | mage | Network scan |
| Patch (apt upgrade) | 4 | 2d6 heal | priest | System update |
| Exploit (metasploit) | 5 | 3d6 damage | mage | Offensive security |
| Firewall (iptables -A) | 3 | Shield 1 turn | priest | Block incoming |

## Act Structure

### Act 1: Reconnaissance (rooms 1-60)

**Goal:** Explore /home, /tmp, /var, learn the system is compromised.

- Start at login_terminal, explore /home/guest
- Discover /tmp is infested with rogue processes
- Find evidence of compromise in /var/log (auth.log shows brute force)
- Discover /home/attacker (hidden directory planted by threat actor)
- Collect basic weapons and armor
- First boss: Null Byte in /dev/null

**Key discoveries:**
- Trojaned binaries in /usr/bin
- Backdoor cron job in /etc/cron.d
- Rogue SSH sessions in /dev/pts

### Act 2: Cleanup (rooms 60-150)

**Goal:** Secure localhost — remove backdoors, defeat rogue processes, harden.

- Clear /etc/cron.d of backdoor entries (defeat Cron Backdoor enemy)
- Remove trojaned binaries from /usr/bin
- Defeat the Rootkit in /usr/lib (requires Grep Tool to reveal it)
- Patch /etc/iptables (restore firewall rules)
- Defeat the OOM Killer in /proc/sys
- Boss: Kernel Panic in /proc/sys/kernel
- Boss: The Daemon King in /root (grants root access)

**Key items earned:**
- Root SSH Key (access to database)
- Admin SSH Key (access to webserver)
- Dev SSH Key (access to buildserver)
- Sudo Token (access to /root)

### Act 3: Counterattack (rooms 150-300)

**Goal:** SSH to remote hosts, trace the attack, destroy the C2 server.

- SSH to webserver: find and remove web shells, discover C2 callback URL
- SSH to database: find exfiltrated data, discover attacker's query logs
- SSH to buildserver: find supply chain injection, discover C2 server address
- SSH to c2_server: final dungeon, defeat the C2 Controller

**Each remote host is a mini-dungeon:**
- 15-25 rooms
- 2-3 unique enemies
- 1 boss
- 1 quest item needed for the next host
- Environmental storytelling via log files and configs

## Progression Gates

| Gate | Requires | Opens |
|------|----------|-------|
| /etc/shadow | (open) | Password knowledge |
| /home/admin | Shadow File or exploit | Admin's home, SSH key |
| /root | Sudo Token | Root's home, root SSH key |
| webserver | Admin SSH Key | Web server exploration |
| database | Root SSH Key | Database exploration |
| buildserver | Dev SSH Key | Build server exploration |
| c2_server | Exploit Kit (from buildserver) | Final dungeon |

## Naming Conventions

Room IDs follow the filesystem path, with `/` replaced by `_` and leading
slash dropped:

```
/home/guest          → home_guest
/var/log/auth.log    → var_log_auth
/etc/cron.d          → etc_cron_d
/dev/null            → dev_null
```

Remote host rooms are prefixed with the hostname:

```
webserver:/var/www    → ws_var_www
database:/var/lib/pg  → db_var_lib_pg
c2_server:/root       → c2_root
```

## Implementation Plan

### Phase 1: Localhost core (100 rooms)

Build the core filesystem: /, /home, /tmp, /var, /etc, /usr, /dev, /proc.
Focus on Act 1 rooms and the critical Act 2 path. Include all tier 1 and
tier 2 enemies, all weapons and armor, and the first three bosses.

**Deliverable:** Playable localhost adventure with ~100 rooms, 3 bosses.

### Phase 2: Localhost expanded (200 rooms)

Add /sys, /opt, /srv, /mnt, /boot, /root. Fill in side rooms, hidden areas,
optional content. Add environmental storytelling (log files that tell the
attack story). Add consumables and additional spells.

**Deliverable:** Full localhost with 200+ rooms, 4 bosses, all items.

### Phase 3: Remote hosts (300 rooms)

Add webserver, database, buildserver, c2_server as SSH-accessible hosts.
Each is a self-contained mini-dungeon. The Exploit Kit from buildserver
unlocks the c2_server final dungeon.

**Deliverable:** Complete game with ~300 rooms, 8 bosses, full story arc.

### Phase 4: Polish

- Balance enemy HP/damage across all three acts
- Ensure every room has a unique description_seed
- Add hidden rooms and optional side quests
- Playtest the full path from login to C2 destruction
- Add victory condition (reaching c2_server root_shell)

## Engine Requirements

The current engine supports everything needed for Phases 1-3. One feature
that would enhance the scenario:

- **SSH as a connection type:** Currently connections are `open`, `locked`,
  `stairway`, `hidden`. Adding an `ssh` type (or using `stairway` for SSH)
  would let the engine display "SSH to webserver" instead of "go up."
  This is cosmetic — `stairway` works mechanically.

No new engine features are required. The scenario expansion is pure content.

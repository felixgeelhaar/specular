# Linux Distribution Package Guide

This guide explains how to get Specular into various Linux distribution package managers.

## Quick Reference

| Distribution | Difficulty | Timeline | Steps |
|-------------|-----------|----------|-------|
| **AUR (Arch)** | ✅ Easy | 1-2 days | [See below](#arch-linux-aur) |
| **Ubuntu PPA** | 🟨 Medium | 1 week | [See below](#ubuntu-ppa) |
| **Copr (Fedora)** | 🟨 Medium | 1 week | [See below](#fedora-copr) |
| **OBS (Multi-distro)** | 🟨 Medium | 2 weeks | [See below](#opensuse-build-service) |
| **Debian Official** | 🟥 Hard | 6+ months | [See below](#official-repositories) |
| **Fedora Official** | 🟥 Hard | 6+ months | [See below](#official-repositories) |

---

## Arch Linux (AUR)

**Status**: ✅ Ready to submit
**Reach**: ~1.5M Arch users + derivatives (Manjaro, EndeavourOS, etc.)

### Prerequisites

1. Create AUR account: https://aur.archlinux.org/register
2. Add SSH key to your AUR account
3. Install required tools:
   ```bash
   sudo pacman -S base-devel git
   ```

### Submission Steps

```bash
# 1. Clone the AUR repository (will be empty initially)
git clone ssh://aur@aur.archlinux.org/specular.git specular-aur
cd specular-aur

# 2. Copy the prepared files
cp ../packaging/aur/PKGBUILD .
cp ../packaging/aur/.SRCINFO .

# 3. Update checksums after v1.1.0 release
# Download the release and generate checksums:
makepkg -g  # This will print the correct checksums

# 4. Update PKGBUILD and .SRCINFO with real checksums

# 5. Test the package locally
makepkg -si

# 6. Commit and push to AUR
git add PKGBUILD .SRCINFO
git commit -m "Initial commit: specular 1.1.0"
git push origin master
```

### After Submission

- Package will be immediately available: https://aur.archlinux.org/packages/specular
- Users install with: `yay -S specular` or `paru -S specular`
- You maintain it by pushing updates to the AUR git repo

---

## Ubuntu PPA

**Status**: Requires setup
**Reach**: Millions of Ubuntu/Debian users

### Prerequisites

1. Create Launchpad account: https://launchpad.net/
2. Import your GPG key to Launchpad
3. Install tools:
   ```bash
   sudo apt-get install devscripts build-essential dput-ng
   ```

### Setup Steps

```bash
# 1. Create PPA on Launchpad
# Go to: https://launchpad.net/~yourusername/+activate-ppa
# Name: specular
# Description: AI-Native Spec and Build Assistant

# 2. Create debian directory structure
cd packaging
mkdir -p ppa/debian

# 3. Create debian/control file (package metadata)
cat > ppa/debian/control << 'EOF'
Source: specular
Section: devel
Priority: optional
Maintainer: Felix Geelhaar <felix@felixgeelhaar.de>
Build-Depends: debhelper-compat (= 13), golang-go (>= 1.22)
Standards-Version: 4.6.0
Homepage: https://github.com/felixgeelhaar/specular

Package: specular
Architecture: amd64 arm64
Depends: ${shlibs:Depends}, ${misc:Depends}, docker.io
Recommends: git
Description: AI-Native Spec and Build Assistant
 Specular provides policy-enforced, AI-powered development workflows
 with multi-provider support and drift detection.
EOF

# 4. Create debian/changelog
cat > ppa/debian/changelog << 'EOF'
specular (1.1.0-1) focal; urgency=medium

  * Initial PPA release
  * Interactive TUI mode
  * Enhanced error system
  * CLI provider protocol

 -- Felix Geelhaar <felix@felixgeelhaar.de>  Thu, 07 Nov 2025 17:00:00 +0100
EOF

# 5. Build source package
cd ppa
debuild -S -sa

# 6. Upload to PPA
dput ppa:yourusername/specular ../specular_1.1.0-1_source.changes
```

### After Upload

- Package builds automatically for multiple Ubuntu versions
- Users add with: `sudo add-apt-repository ppa:yourusername/specular`
- Then: `sudo apt update && sudo apt install specular`

---

## Fedora Copr

**Status**: Requires setup
**Reach**: Fedora, RHEL, CentOS users

### Prerequisites

1. Create Fedora account: https://accounts.fedoraproject.org/
2. Enable Copr: https://copr.fedorainfracloud.org/
3. Install tools:
   ```bash
   sudo dnf install copr-cli rpm-build
   ```

### Setup Steps

```bash
# 1. Create Copr project via web UI
# Go to: https://copr.fedorainfracloud.org/coprs/add/
# Name: specular
# Description: AI-Native Spec and Build Assistant
# Instructions: Select Fedora versions to build for

# 2. Create specular.spec file
cat > packaging/copr/specular.spec << 'EOF'
Name:           specular
Version:        1.1.0
Release:        1%{?dist}
Summary:        AI-Native Spec and Build Assistant

License:        MIT
URL:            https://github.com/felixgeelhaar/specular
Source0:        https://github.com/felixgeelhaar/specular/archive/v%{version}.tar.gz

BuildRequires:  golang >= 1.22
Requires:       docker

%description
Specular provides policy-enforced, AI-powered development workflows
with multi-provider support and drift detection.

%prep
%setup -q

%build
make build

%install
install -Dm755 specular %{buildroot}%{_bindir}/specular

%files
%license LICENSE
%doc README.md
%{_bindir}/specular

%changelog
* Thu Nov 07 2025 Felix Geelhaar <felix@felixgeelhaar.de> - 1.1.0-1
- Initial Copr release
EOF

# 3. Build and upload
copr-cli build yourusername/specular packaging/copr/specular.spec
```

### After Setup

- Users enable with: `sudo dnf copr enable yourusername/specular`
- Then: `sudo dnf install specular`

---

## OpenSUSE Build Service (OBS)

**Status**: Requires setup
**Reach**: Multi-distro (openSUSE, Fedora, Debian, Ubuntu, Arch, etc.)

### Why OBS?

OBS builds packages for **multiple distributions simultaneously** from a single configuration. One upload → packages for 10+ distros.

### Prerequisites

1. Create account: https://build.opensuse.org/
2. Install `osc` tool:
   ```bash
   # On openSUSE
   sudo zypper install osc

   # On other distros
   pip install osc
   ```

### Setup Steps

```bash
# 1. Configure osc
osc config set apiurl https://api.opensuse.org
osc config set user yourusername

# 2. Create new package
osc meta pkg -e home:yourusername specular

# 3. Checkout package
osc checkout home:yourusername/specular
cd home:yourusername/specular

# 4. Add source files
osc add specular-1.1.0.tar.gz
osc add specular.spec

# 5. Commit and build
osc commit
osc results  # Watch build progress
```

### After Setup

- OBS builds for all configured distros automatically
- Users can add your repo and install

---

## Official Repositories

### Requirements for Official Inclusion

All major distributions require:

1. **Active Maintenance**
   - Regular updates
   - Security patch responsiveness
   - Bug fix commitment

2. **Package Quality**
   - Meets distribution packaging guidelines
   - No policy violations
   - Clean lintian/rpmlint results
   - Proper dependencies

3. **Community Support**
   - User adoption
   - Active issue resolution
   - Documentation
   - Responsive maintainer

4. **Stability**
   - Proven track record (6-12 months)
   - No critical bugs
   - Reliable releases

### Debian Official

**Timeline**: 6-12 months minimum

1. Join Debian: https://www.debian.org/devel/join/newmaint
2. Find sponsor: https://mentors.debian.net/
3. Package review process
4. NEW queue approval
5. Testing → Unstable → Stable migration

**Resources**:
- Debian New Maintainers Guide: https://www.debian.org/doc/manuals/maint-guide/
- Debian Policy: https://www.debian.org/doc/debian-policy/

### Fedora Official

**Timeline**: 3-6 months minimum

1. Create Fedora Account System (FAS) account
2. Join packaging group
3. Submit package review: https://bugzilla.redhat.com
4. Find sponsor for review
5. Pass review process
6. Maintain package in Fedora

**Resources**:
- Fedora Packaging Guidelines: https://docs.fedoraproject.org/en-US/packaging-guidelines/
- Package Review Process: https://fedoraproject.org/wiki/Package_Review_Process

---

## Recommended Strategy

### Phase 1: Community Repos (Now - 1 Month)

✅ **Completed:**
- GitHub Releases (DEB/RPM/APK)
- Homebrew tap
- Docker images

🎯 **Next Steps:**
1. **Submit to AUR** (highest ROI, easiest)
2. **Create Ubuntu PPA** (large user base)
3. **Set up Copr** (Fedora ecosystem)

### Phase 2: Multi-Distro (1-3 Months)

4. **OpenSUSE Build Service** (builds for many distros)
5. **Build adoption** (users, stars, issues resolved)
6. **Establish stability** (regular releases, no critical bugs)

### Phase 3: Official Inclusion (6+ Months)

7. **Apply to Debian** (largest reach)
8. **Apply to Fedora** (enterprise adoption)
9. **Community maintenance** (long-term commitment)

---

## Maintenance Checklist

For each new release:

- [ ] Update version in PKGBUILD
- [ ] Update checksums in PKGBUILD
- [ ] Regenerate .SRCINFO: `makepkg --printsrcinfo > .SRCINFO`
- [ ] Update PPA debian/changelog
- [ ] Rebuild Copr packages
- [ ] Update OBS service files
- [ ] Test installation on each platform
- [ ] Update documentation

---

## Support Contacts

- **AUR**: https://wiki.archlinux.org/title/AUR_submission_guidelines
- **Ubuntu PPA**: https://help.launchpad.net/Packaging/PPA
- **Fedora Copr**: https://docs.pagure.org/copr.copr/user_documentation.html
- **OBS**: https://openbuildservice.org/help/manuals/obs-user-guide/

---

## Current Status

| Platform | Status | Link |
|----------|--------|------|
| GitHub Releases | ✅ Live | https://github.com/felixgeelhaar/specular/releases |
| Homebrew | ✅ Live | `brew install felixgeelhaar/tap/specular` |
| Docker | ✅ Live | `ghcr.io/felixgeelhaar/specular` |
| AUR | 📦 Ready | Files in `packaging/aur/` |
| Ubuntu PPA | ⏳ TODO | Need Launchpad setup |
| Fedora Copr | ⏳ TODO | Need Copr setup |
| OBS | ⏳ TODO | Need OBS account |
| Debian Official | 🎯 Future | Requires 6+ months stability |
| Fedora Official | 🎯 Future | Requires 3+ months stability |

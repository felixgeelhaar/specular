Name:           specular
Version:        %{version}
Release:        1%{?dist}
Summary:        AI-native spec and build assistant
License:        Apache-2.0
URL:            https://github.com/felixgeelhaar/specular
Source0:        %{name}-%{version}.tar.gz
BuildArch:      x86_64
BuildRequires:  golang
Requires(post): /usr/bin/specular

%description
Specular is a specification-first, policy-aware workflow tool powered by AI.

%prep
%setup -q

%build
go build -ldflags "-s -w" -o specular ./cmd/specular

%install
install -D specular %{buildroot}/usr/bin/specular

%files
/usr/bin/specular

%changelog
* TBD Release workflow

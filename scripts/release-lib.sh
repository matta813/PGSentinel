#!/bin/sh

semver_pattern='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9A-Za-z-][0-9A-Za-z-]*)\.)*(0|[1-9A-Za-z-][0-9A-Za-z-]*))?$'

trim_version() {
  tr -d '\r\n\t ' < "$1"
}

validate_semver() {
  printf '%s\n' "$1" | grep -Eq "$semver_pattern"
}

is_prerelease() {
  case "$1" in *-*) return 0 ;; *) return 1 ;; esac
}

# Prints -1, 0, or 1 for A < B, A = B, or A > B according to SemVer precedence.
compare_semver() {
  awk -v a="$1" -v b="$2" '
  function splitver(v,out, core,pos) {
    pos=index(v,"-"); core=(pos?substr(v,1,pos-1):v)
    split(core,out,"."); out[4]=(pos?substr(v,pos+1):"")
  }
  function numcmp(x,y) {
    if(length(x)!=length(y)) return (length(x)<length(y)?-1:1)
    return (x<y?-1:(x>y?1:0))
  }
  function identcmp(x,y, xa,ya,nx,ny) {
    nx=(x ~ /^[0-9]+$/); ny=(y ~ /^[0-9]+$/)
    if(nx && ny) return numcmp(x,y)
    if(nx != ny) return (nx?-1:1)
    return (x<y?-1:(x>y?1:0))
  }
  BEGIN {
    splitver(a,x); splitver(b,y)
    for(i=1;i<=3;i++) {c=numcmp(x[i],y[i]); if(c!=0){print c;exit}}
    if(x[4]=="" || y[4]=="") {print(x[4]==y[4]?0:(x[4]==""?1:-1)); exit}
    nx=split(x[4],px,"."); ny=split(y[4],py,"."); n=(nx>ny?nx:ny)
    for(i=1;i<=n;i++) {
      if(i>nx){print -1;exit}; if(i>ny){print 1;exit}
      c=identcmp(px[i],py[i]); if(c!=0){print c;exit}
    }
    print 0
  }'
}

# Prints the highest valid PGSentinel release tag according to SemVer precedence.
latest_release_tag() {
  latest=
  for candidate in $(git tag --list 'v*'); do
    version=${candidate#v}
    [ "v$version" = "$candidate" ] || continue
    validate_semver "$version" || continue
    if [ -z "$latest" ] || [ "$(compare_semver "$version" "${latest#v}")" -gt 0 ]; then
      latest=$candidate
    fi
  done
  printf '%s\n' "$latest"
}

# Prints the highest valid tag whose version is strictly lower than CURRENT.
previous_release_tag() {
  current=${1#v}
  validate_semver "$current" || {
    echo "Invalid current release version: $1" >&2
    return 1
  }
  previous=
  for candidate in $(git tag --list 'v*'); do
    version=${candidate#v}
    [ "v$version" = "$candidate" ] || continue
    validate_semver "$version" || continue
    [ "$(compare_semver "$version" "$current")" -lt 0 ] || continue
    if [ -z "$previous" ] || [ "$(compare_semver "$version" "${previous#v}")" -gt 0 ]; then
      previous=$candidate
    fi
  done
  printf '%s\n' "$previous"
}

# Resolves lightweight or annotated TAG to its immutable commit SHA.
tag_commit_sha() {
  tag=$1
  git show-ref --verify --quiet "refs/tags/$tag" || {
    echo "Release tag does not exist: $tag" >&2
    return 1
  }
  git rev-parse --verify "$tag^{commit}"
}

release_version_at_sha() {
  source_sha=$1
  raw=$(git show "$source_sha:RELEASE") || return 1
  version=$(printf '%s' "$raw" | tr -d '\r\n\t ')
  [ "$raw" = "$version" ] || {
    echo "RELEASE at $source_sha is not a single clean version line" >&2
    return 1
  }
  validate_semver "$version" || {
    echo "Invalid RELEASE version at $source_sha: $version" >&2
    return 1
  }
  printf '%s\n' "$version"
}

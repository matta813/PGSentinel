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
  function splitver(v,out, p,n,i) {
    n=split(v,p,"-"); split(p[1],out,"."); out[4]=(n>1?p[2]:"")
  }
  function identcmp(x,y, xa,ya,nx,ny) {
    nx=(x ~ /^[0-9]+$/); ny=(y ~ /^[0-9]+$/)
    if(nx && ny) return (x+0<y+0?-1:(x+0>y+0?1:0))
    if(nx != ny) return (nx?-1:1)
    return (x<y?-1:(x>y?1:0))
  }
  BEGIN {
    splitver(a,x); splitver(b,y)
    for(i=1;i<=3;i++) if(x[i]+0 != y[i]+0) {print(x[i]+0<y[i]+0?-1:1); exit}
    if(x[4]=="" || y[4]=="") {print(x[4]==y[4]?0:(x[4]==""?1:-1)); exit}
    nx=split(x[4],px,"."); ny=split(y[4],py,"."); n=(nx>ny?nx:ny)
    for(i=1;i<=n;i++) {
      if(i>nx){print -1;exit}; if(i>ny){print 1;exit}
      c=identcmp(px[i],py[i]); if(c!=0){print c;exit}
    }
    print 0
  }'
}

#!/usr/bin/env sh

if test "$#" -ne 3; then
	echo "Usage: $0 package varname [--compress]"
	echo "Base64 encodes stdin and sets val to varname, outputs Go on stdout"
	exit 1
fi

cat << EOF
package $1

const $2 = "$( (test "$3" = "--compress" && gzip -qc8 || cat) | base64 -w0 )"
EOF

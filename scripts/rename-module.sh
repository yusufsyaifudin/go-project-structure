#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'EOF'
Usage: scripts/rename-module.sh OLD_MODULE NEW_MODULE OLD_NAME NEW_NAME [OLD_PACKAGE NEW_PACKAGE]

Replaces module/import paths and project-name references in tracked text files.
Run from the repository root, then review the diff and run `go mod tidy`.
EOF
}

if [[ $# -ne 4 && $# -ne 6 ]]; then
	usage >&2
	exit 2
fi

old_module=$1
new_module=$2
old_name=$3
new_name=$4
old_package=${5:-}
new_package=${6:-}

if [[ $# -eq 5 ]]; then
	usage >&2
	exit 2
fi

if [[ $# -eq 6 && ( -z $old_package || -z $new_package ) ]]; then
	usage >&2
	exit 2
fi

if [[ -z $old_module || -z $new_module || -z $old_name || -z $new_name ]]; then
	usage >&2
	exit 2
fi

if [[ ! -f go.mod || ! -d .git ]]; then
	echo "run this script from the repository root" >&2
	exit 1
fi

if ! grep -Fqx "module $old_module" go.mod; then
	echo "go.mod does not declare expected module: $old_module" >&2
	exit 1
fi

sed_pattern_escape() {
	printf '%s' "$1" | sed 's/[.[\*^$\\]/\\&/g; s/|/\\|/g'
}

sed_replacement_escape() {
	printf '%s' "$1" | sed 's/[&|\\]/\\&/g'
}

old_module_re=$(sed_pattern_escape "$old_module")
new_module_replacement=$(sed_replacement_escape "$new_module")
old_name_re=$(sed_pattern_escape "$old_name")
new_name_replacement=$(sed_replacement_escape "$new_name")
if [[ -n $old_package ]]; then
	old_package_re=$(sed_pattern_escape "$old_package")
	new_package_replacement=$(sed_replacement_escape "$new_package")
else
	old_package_re=
	new_package_replacement=
fi

while IFS= read -r -d '' file; do
	if [[ $file == scripts/rename-module.sh ]]; then
		continue
	fi
	if ! file "$file" | grep -q 'text'; then
		continue
	fi

	if grep -Fq "$old_module" "$file" || grep -Fq "$old_name" "$file" || \
		{ [[ -n $old_package ]] && grep -Fq "$old_package" "$file"; }; then
		sed_args=(
			-e "s|$old_module_re|$new_module_replacement|g"
			-e "s|$old_name_re|$new_name_replacement|g"
		)
		if [[ -n $old_package_re ]]; then
			sed_args+=( -e "s|$old_package_re|$new_package_replacement|g" )
		fi
		sed -i '' "${sed_args[@]}" "$file"
	fi
done < <(git ls-files -z --cached --others --exclude-standard)

echo "Updated module and project references. Review with: git diff"

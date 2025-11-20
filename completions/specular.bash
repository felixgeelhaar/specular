# bash completion for specular                             -*- shell-script -*-

__specular_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE:-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# Homebrew on Macs have version 1.3 of bash-completion which doesn't include
# _init_completion. This is a very minimal version of that function.
__specular_init_completion()
{
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
}

__specular_index_of_word()
{
    local w word=$1
    shift
    index=0
    for w in "$@"; do
        [[ $w = "$word" ]] && return
        index=$((index+1))
    done
    index=-1
}

__specular_contains_word()
{
    local w word=$1; shift
    for w in "$@"; do
        [[ $w = "$word" ]] && return
    done
    return 1
}

__specular_handle_go_custom_completion()
{
    __specular_debug "${FUNCNAME[0]}: cur is ${cur}, words[*] is ${words[*]}, #words[@] is ${#words[@]}"

    local shellCompDirectiveError=1
    local shellCompDirectiveNoSpace=2
    local shellCompDirectiveNoFileComp=4
    local shellCompDirectiveFilterFileExt=8
    local shellCompDirectiveFilterDirs=16

    local out requestComp lastParam lastChar comp directive args

    # Prepare the command to request completions for the program.
    # Calling ${words[0]} instead of directly specular allows handling aliases
    args=("${words[@]:1}")
    # Disable ActiveHelp which is not supported for bash completion v1
    requestComp="SPECULAR_ACTIVE_HELP=0 ${words[0]} __completeNoDesc ${args[*]}"

    lastParam=${words[$((${#words[@]}-1))]}
    lastChar=${lastParam:$((${#lastParam}-1)):1}
    __specular_debug "${FUNCNAME[0]}: lastParam ${lastParam}, lastChar ${lastChar}"

    if [ -z "${cur}" ] && [ "${lastChar}" != "=" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go method.
        __specular_debug "${FUNCNAME[0]}: Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __specular_debug "${FUNCNAME[0]}: calling ${requestComp}"
    # Use eval to handle any environment variables and such
    out=$(eval "${requestComp}" 2>/dev/null)

    # Extract the directive integer at the very end of the output following a colon (:)
    directive=${out##*:}
    # Remove the directive
    out=${out%:*}
    if [ "${directive}" = "${out}" ]; then
        # There is not directive specified
        directive=0
    fi
    __specular_debug "${FUNCNAME[0]}: the completion directive is: ${directive}"
    __specular_debug "${FUNCNAME[0]}: the completions are: ${out}"

    if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        # Error code.  No completion.
        __specular_debug "${FUNCNAME[0]}: received error from custom completion go code"
        return
    else
        if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __specular_debug "${FUNCNAME[0]}: activating no space"
                compopt -o nospace
            fi
        fi
        if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __specular_debug "${FUNCNAME[0]}: activating no file completion"
                compopt +o default
            fi
        fi
    fi

    if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
        # File extension filtering
        local fullFilter filter filteringCmd
        # Do not use quotes around the $out variable or else newline
        # characters will be kept.
        for filter in ${out}; do
            fullFilter+="$filter|"
        done

        filteringCmd="_filedir $fullFilter"
        __specular_debug "File filtering command: $filteringCmd"
        $filteringCmd
    elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
        # File completion for directories only
        local subdir
        # Use printf to strip any trailing newline
        subdir=$(printf "%s" "${out}")
        if [ -n "$subdir" ]; then
            __specular_debug "Listing directories in $subdir"
            __specular_handle_subdirs_in_dir_flag "$subdir"
        else
            __specular_debug "Listing directories in ."
            _filedir -d
        fi
    else
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${out}" -- "$cur")
    fi
}

__specular_handle_reply()
{
    __specular_debug "${FUNCNAME[0]}"
    local comp
    case $cur in
        -*)
            if [[ $(type -t compopt) = "builtin" ]]; then
                compopt -o nospace
            fi
            local allflags
            if [ ${#must_have_one_flag[@]} -ne 0 ]; then
                allflags=("${must_have_one_flag[@]}")
            else
                allflags=("${flags[*]} ${two_word_flags[*]}")
            fi
            while IFS='' read -r comp; do
                COMPREPLY+=("$comp")
            done < <(compgen -W "${allflags[*]}" -- "$cur")
            if [[ $(type -t compopt) = "builtin" ]]; then
                [[ "${COMPREPLY[0]}" == *= ]] || compopt +o nospace
            fi

            # complete after --flag=abc
            if [[ $cur == *=* ]]; then
                if [[ $(type -t compopt) = "builtin" ]]; then
                    compopt +o nospace
                fi

                local index flag
                flag="${cur%=*}"
                __specular_index_of_word "${flag}" "${flags_with_completion[@]}"
                COMPREPLY=()
                if [[ ${index} -ge 0 ]]; then
                    PREFIX=""
                    cur="${cur#*=}"
                    ${flags_completion[${index}]}
                    if [ -n "${ZSH_VERSION:-}" ]; then
                        # zsh completion needs --flag= prefix
                        eval "COMPREPLY=( \"\${COMPREPLY[@]/#/${flag}=}\" )"
                    fi
                fi
            fi

            if [[ -z "${flag_parsing_disabled}" ]]; then
                # If flag parsing is enabled, we have completed the flags and can return.
                # If flag parsing is disabled, we may not know all (or any) of the flags, so we fallthrough
                # to possibly call handle_go_custom_completion.
                return 0;
            fi
            ;;
    esac

    # check if we are handling a flag with special work handling
    local index
    __specular_index_of_word "${prev}" "${flags_with_completion[@]}"
    if [[ ${index} -ge 0 ]]; then
        ${flags_completion[${index}]}
        return
    fi

    # we are parsing a flag and don't have a special handler, no completion
    if [[ ${cur} != "${words[cword]}" ]]; then
        return
    fi

    local completions
    completions=("${commands[@]}")
    if [[ ${#must_have_one_noun[@]} -ne 0 ]]; then
        completions+=("${must_have_one_noun[@]}")
    elif [[ -n "${has_completion_function}" ]]; then
        # if a go completion function is provided, defer to that function
        __specular_handle_go_custom_completion
    fi
    if [[ ${#must_have_one_flag[@]} -ne 0 ]]; then
        completions+=("${must_have_one_flag[@]}")
    fi
    while IFS='' read -r comp; do
        COMPREPLY+=("$comp")
    done < <(compgen -W "${completions[*]}" -- "$cur")

    if [[ ${#COMPREPLY[@]} -eq 0 && ${#noun_aliases[@]} -gt 0 && ${#must_have_one_noun[@]} -ne 0 ]]; then
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${noun_aliases[*]}" -- "$cur")
    fi

    if [[ ${#COMPREPLY[@]} -eq 0 ]]; then
        if declare -F __specular_custom_func >/dev/null; then
            # try command name qualified custom func
            __specular_custom_func
        else
            # otherwise fall back to unqualified for compatibility
            declare -F __custom_func >/dev/null && __custom_func
        fi
    fi

    # available in bash-completion >= 2, not always present on macOS
    if declare -F __ltrim_colon_completions >/dev/null; then
        __ltrim_colon_completions "$cur"
    fi

    # If there is only 1 completion and it is a flag with an = it will be completed
    # but we don't want a space after the =
    if [[ "${#COMPREPLY[@]}" -eq "1" ]] && [[ $(type -t compopt) = "builtin" ]] && [[ "${COMPREPLY[0]}" == --*= ]]; then
       compopt -o nospace
    fi
}

# The arguments should be in the form "ext1|ext2|extn"
__specular_handle_filename_extension_flag()
{
    local ext="$1"
    _filedir "@(${ext})"
}

__specular_handle_subdirs_in_dir_flag()
{
    local dir="$1"
    pushd "${dir}" >/dev/null 2>&1 && _filedir -d && popd >/dev/null 2>&1 || return
}

__specular_handle_flag()
{
    __specular_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    # if a command required a flag, and we found it, unset must_have_one_flag()
    local flagname=${words[c]}
    local flagvalue=""
    # if the word contained an =
    if [[ ${words[c]} == *"="* ]]; then
        flagvalue=${flagname#*=} # take in as flagvalue after the =
        flagname=${flagname%=*} # strip everything after the =
        flagname="${flagname}=" # but put the = back
    fi
    __specular_debug "${FUNCNAME[0]}: looking for ${flagname}"
    if __specular_contains_word "${flagname}" "${must_have_one_flag[@]}"; then
        must_have_one_flag=()
    fi

    # if you set a flag which only applies to this command, don't show subcommands
    if __specular_contains_word "${flagname}" "${local_nonpersistent_flags[@]}"; then
      commands=()
    fi

    # keep flag value with flagname as flaghash
    # flaghash variable is an associative array which is only supported in bash > 3.
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        if [ -n "${flagvalue}" ] ; then
            flaghash[${flagname}]=${flagvalue}
        elif [ -n "${words[ $((c+1)) ]}" ] ; then
            flaghash[${flagname}]=${words[ $((c+1)) ]}
        else
            flaghash[${flagname}]="true" # pad "true" for bool flag
        fi
    fi

    # skip the argument to a two word flag
    if [[ ${words[c]} != *"="* ]] && __specular_contains_word "${words[c]}" "${two_word_flags[@]}"; then
        __specular_debug "${FUNCNAME[0]}: found a flag ${words[c]}, skip the next argument"
        c=$((c+1))
        # if we are looking for a flags value, don't show commands
        if [[ $c -eq $cword ]]; then
            commands=()
        fi
    fi

    c=$((c+1))

}

__specular_handle_noun()
{
    __specular_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    if __specular_contains_word "${words[c]}" "${must_have_one_noun[@]}"; then
        must_have_one_noun=()
    elif __specular_contains_word "${words[c]}" "${noun_aliases[@]}"; then
        must_have_one_noun=()
    fi

    nouns+=("${words[c]}")
    c=$((c+1))
}

__specular_handle_command()
{
    __specular_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    local next_command
    if [[ -n ${last_command} ]]; then
        next_command="_${last_command}_${words[c]//:/__}"
    else
        if [[ $c -eq 0 ]]; then
            next_command="_specular_root_command"
        else
            next_command="_${words[c]//:/__}"
        fi
    fi
    c=$((c+1))
    __specular_debug "${FUNCNAME[0]}: looking for ${next_command}"
    declare -F "$next_command" >/dev/null && $next_command
}

__specular_handle_word()
{
    if [[ $c -ge $cword ]]; then
        __specular_handle_reply
        return
    fi
    __specular_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"
    if [[ "${words[c]}" == -* ]]; then
        __specular_handle_flag
    elif __specular_contains_word "${words[c]}" "${commands[@]}"; then
        __specular_handle_command
    elif [[ $c -eq 0 ]]; then
        __specular_handle_command
    elif __specular_contains_word "${words[c]}" "${command_aliases[@]}"; then
        # aliashash variable is an associative array which is only supported in bash > 3.
        if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
            words[c]=${aliashash[${words[c]}]}
            __specular_handle_command
        else
            __specular_handle_noun
        fi
    else
        __specular_handle_noun
    fi
    __specular_handle_word
}

_specular_approvals_list()
{
    last_command="specular_approvals_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_approvals_pending()
{
    last_command="specular_approvals_pending"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_approvals()
{
    last_command="specular_approvals"

    command_aliases=()

    commands=()
    commands+=("list")
    commands+=("pending")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_approve()
{
    last_command="specular_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--message=")
    two_word_flags+=("--message")
    local_nonpersistent_flags+=("--message")
    local_nonpersistent_flags+=("--message=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auth_login()
{
    last_command="specular_auth_login"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--email=")
    two_word_flags+=("--email")
    local_nonpersistent_flags+=("--email")
    local_nonpersistent_flags+=("--email=")
    flags+=("--password=")
    two_word_flags+=("--password")
    local_nonpersistent_flags+=("--password")
    local_nonpersistent_flags+=("--password=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auth_logout()
{
    last_command="specular_auth_logout"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auth_register()
{
    last_command="specular_auth_register"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--email=")
    two_word_flags+=("--email")
    local_nonpersistent_flags+=("--email")
    local_nonpersistent_flags+=("--email=")
    flags+=("--first-name=")
    two_word_flags+=("--first-name")
    local_nonpersistent_flags+=("--first-name")
    local_nonpersistent_flags+=("--first-name=")
    flags+=("--last-name=")
    two_word_flags+=("--last-name")
    local_nonpersistent_flags+=("--last-name")
    local_nonpersistent_flags+=("--last-name=")
    flags+=("--password=")
    two_word_flags+=("--password")
    local_nonpersistent_flags+=("--password")
    local_nonpersistent_flags+=("--password=")
    flags+=("--username=")
    two_word_flags+=("--username")
    local_nonpersistent_flags+=("--username")
    local_nonpersistent_flags+=("--username=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auth_status()
{
    last_command="specular_auth_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auth()
{
    last_command="specular_auth"

    command_aliases=()

    commands=()
    commands+=("login")
    commands+=("logout")
    commands+=("register")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auto_explain()
{
    last_command="specular_auto_explain"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auto_history()
{
    last_command="specular_auto_history"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auto_resume()
{
    last_command="specular_auto_resume"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auto_rollback()
{
    last_command="specular_auto_rollback"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--list")
    local_nonpersistent_flags+=("--list")
    flags+=("--to=")
    two_word_flags+=("--to")
    local_nonpersistent_flags+=("--to")
    local_nonpersistent_flags+=("--to=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auto_verify()
{
    last_command="specular_auto_verify"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allowed-identity=")
    two_word_flags+=("--allowed-identity")
    local_nonpersistent_flags+=("--allowed-identity")
    local_nonpersistent_flags+=("--allowed-identity=")
    flags+=("--max-age=")
    two_word_flags+=("--max-age")
    local_nonpersistent_flags+=("--max-age")
    local_nonpersistent_flags+=("--max-age=")
    flags+=("--output=")
    two_word_flags+=("--output")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--require-clean-git")
    local_nonpersistent_flags+=("--require-clean-git")
    flags+=("--verify-hashes")
    local_nonpersistent_flags+=("--verify-hashes")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_auto()
{
    last_command="specular_auto"

    command_aliases=()

    commands=()
    commands+=("explain")
    commands+=("history")
    commands+=("resume")
    commands+=("rollback")
    commands+=("verify")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--attest")
    local_nonpersistent_flags+=("--attest")
    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--include-dependencies")
    local_nonpersistent_flags+=("--include-dependencies")
    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--list-profiles")
    local_nonpersistent_flags+=("--list-profiles")
    flags+=("--max-cost=")
    two_word_flags+=("--max-cost")
    local_nonpersistent_flags+=("--max-cost")
    local_nonpersistent_flags+=("--max-cost=")
    flags+=("--max-cost-per-task=")
    two_word_flags+=("--max-cost-per-task")
    local_nonpersistent_flags+=("--max-cost-per-task")
    local_nonpersistent_flags+=("--max-cost-per-task=")
    flags+=("--max-retries=")
    two_word_flags+=("--max-retries")
    local_nonpersistent_flags+=("--max-retries")
    local_nonpersistent_flags+=("--max-retries=")
    flags+=("--max-steps=")
    two_word_flags+=("--max-steps")
    local_nonpersistent_flags+=("--max-steps")
    local_nonpersistent_flags+=("--max-steps=")
    flags+=("--no-approval")
    local_nonpersistent_flags+=("--no-approval")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--profile")
    local_nonpersistent_flags+=("--profile=")
    local_nonpersistent_flags+=("-p")
    flags+=("--resume=")
    two_word_flags+=("--resume")
    local_nonpersistent_flags+=("--resume")
    local_nonpersistent_flags+=("--resume=")
    flags+=("--save-patches")
    local_nonpersistent_flags+=("--save-patches")
    flags+=("--scope=")
    two_word_flags+=("--scope")
    two_word_flags+=("-s")
    local_nonpersistent_flags+=("--scope")
    local_nonpersistent_flags+=("--scope=")
    local_nonpersistent_flags+=("-s")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--trace")
    local_nonpersistent_flags+=("--trace")
    flags+=("--tui")
    local_nonpersistent_flags+=("--tui")
    flags+=("--verbose")
    flags+=("-v")
    local_nonpersistent_flags+=("--verbose")
    local_nonpersistent_flags+=("-v")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_build_approve()
{
    last_command="specular_build_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--manifest-dir=")
    two_word_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_build_explain()
{
    last_command="specular_build_explain"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--manifest-dir=")
    two_word_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_build_run()
{
    last_command="specular_build_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cache-dir=")
    two_word_flags+=("--cache-dir")
    local_nonpersistent_flags+=("--cache-dir")
    local_nonpersistent_flags+=("--cache-dir=")
    flags+=("--cache-max-age=")
    two_word_flags+=("--cache-max-age")
    local_nonpersistent_flags+=("--cache-max-age")
    local_nonpersistent_flags+=("--cache-max-age=")
    flags+=("--checkpoint-dir=")
    two_word_flags+=("--checkpoint-dir")
    local_nonpersistent_flags+=("--checkpoint-dir")
    local_nonpersistent_flags+=("--checkpoint-dir=")
    flags+=("--checkpoint-id=")
    two_word_flags+=("--checkpoint-id")
    local_nonpersistent_flags+=("--checkpoint-id")
    local_nonpersistent_flags+=("--checkpoint-id=")
    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--enable-cache")
    local_nonpersistent_flags+=("--enable-cache")
    flags+=("--fail-on=")
    two_word_flags+=("--fail-on")
    local_nonpersistent_flags+=("--fail-on")
    local_nonpersistent_flags+=("--fail-on=")
    flags+=("--feature=")
    two_word_flags+=("--feature")
    local_nonpersistent_flags+=("--feature")
    local_nonpersistent_flags+=("--feature=")
    flags+=("--keep-checkpoint")
    local_nonpersistent_flags+=("--keep-checkpoint")
    flags+=("--manifest-dir=")
    two_word_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--resume")
    local_nonpersistent_flags+=("--resume")
    flags+=("--verbose")
    local_nonpersistent_flags+=("--verbose")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_build_verify()
{
    last_command="specular_build_verify"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_build()
{
    last_command="specular_build"

    command_aliases=()

    commands=()
    commands+=("approve")
    commands+=("explain")
    commands+=("run")
    commands+=("verify")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--manifest-dir=")
    two_word_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir")
    local_nonpersistent_flags+=("--manifest-dir=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_apply()
{
    last_command="specular_bundle_apply"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--exclude=")
    two_word_flags+=("--exclude")
    local_nonpersistent_flags+=("--exclude")
    local_nonpersistent_flags+=("--exclude=")
    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--target-dir=")
    two_word_flags+=("--target-dir")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--target-dir")
    local_nonpersistent_flags+=("--target-dir=")
    local_nonpersistent_flags+=("-t")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_approval-status()
{
    last_command="specular_bundle_approval-status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--approvals=")
    two_word_flags+=("--approvals")
    two_word_flags+=("-a")
    local_nonpersistent_flags+=("--approvals")
    local_nonpersistent_flags+=("--approvals=")
    local_nonpersistent_flags+=("-a")
    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--required-roles=")
    two_word_flags+=("--required-roles")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--required-roles")
    local_nonpersistent_flags+=("--required-roles=")
    local_nonpersistent_flags+=("-r")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_flag+=("--approvals=")
    must_have_one_flag+=("-a")
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_approve()
{
    last_command="specular_bundle_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--comment=")
    two_word_flags+=("--comment")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--comment")
    local_nonpersistent_flags+=("--comment=")
    local_nonpersistent_flags+=("-c")
    flags+=("--key-path=")
    two_word_flags+=("--key-path")
    two_word_flags+=("-k")
    local_nonpersistent_flags+=("--key-path")
    local_nonpersistent_flags+=("--key-path=")
    local_nonpersistent_flags+=("-k")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--role=")
    two_word_flags+=("--role")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--role")
    local_nonpersistent_flags+=("--role=")
    local_nonpersistent_flags+=("-r")
    flags+=("--signature-type=")
    two_word_flags+=("--signature-type")
    local_nonpersistent_flags+=("--signature-type")
    local_nonpersistent_flags+=("--signature-type=")
    flags+=("--user=")
    two_word_flags+=("--user")
    two_word_flags+=("-u")
    local_nonpersistent_flags+=("--user")
    local_nonpersistent_flags+=("--user=")
    local_nonpersistent_flags+=("-u")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_flag+=("--role=")
    must_have_one_flag+=("-r")
    must_have_one_flag+=("--user=")
    must_have_one_flag+=("-u")
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_create()
{
    last_command="specular_bundle_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--attest")
    local_nonpersistent_flags+=("--attest")
    flags+=("--attest-format=")
    two_word_flags+=("--attest-format")
    local_nonpersistent_flags+=("--attest-format")
    local_nonpersistent_flags+=("--attest-format=")
    flags+=("--governance-level=")
    two_word_flags+=("--governance-level")
    two_word_flags+=("-g")
    local_nonpersistent_flags+=("--governance-level")
    local_nonpersistent_flags+=("--governance-level=")
    local_nonpersistent_flags+=("-g")
    flags+=("--include=")
    two_word_flags+=("--include")
    two_word_flags+=("-i")
    local_nonpersistent_flags+=("--include")
    local_nonpersistent_flags+=("--include=")
    local_nonpersistent_flags+=("-i")
    flags+=("--lock=")
    two_word_flags+=("--lock")
    local_nonpersistent_flags+=("--lock")
    local_nonpersistent_flags+=("--lock=")
    flags+=("--metadata=")
    two_word_flags+=("--metadata")
    two_word_flags+=("-m")
    local_nonpersistent_flags+=("--metadata")
    local_nonpersistent_flags+=("--metadata=")
    local_nonpersistent_flags+=("-m")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    local_nonpersistent_flags+=("-p")
    flags+=("--require-approval=")
    two_word_flags+=("--require-approval")
    two_word_flags+=("-a")
    local_nonpersistent_flags+=("--require-approval")
    local_nonpersistent_flags+=("--require-approval=")
    local_nonpersistent_flags+=("-a")
    flags+=("--routing=")
    two_word_flags+=("--routing")
    local_nonpersistent_flags+=("--routing")
    local_nonpersistent_flags+=("--routing=")
    flags+=("--spec=")
    two_word_flags+=("--spec")
    local_nonpersistent_flags+=("--spec")
    local_nonpersistent_flags+=("--spec=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_diff()
{
    last_command="specular_bundle_diff"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--quiet")
    flags+=("-q")
    local_nonpersistent_flags+=("--quiet")
    local_nonpersistent_flags+=("-q")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_gate()
{
    last_command="specular_bundle_gate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--offline")
    local_nonpersistent_flags+=("--offline")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--require-approvals")
    local_nonpersistent_flags+=("--require-approvals")
    flags+=("--strict")
    local_nonpersistent_flags+=("--strict")
    flags+=("--trusted-key=")
    two_word_flags+=("--trusted-key")
    local_nonpersistent_flags+=("--trusted-key")
    local_nonpersistent_flags+=("--trusted-key=")
    flags+=("--verify-attestation")
    local_nonpersistent_flags+=("--verify-attestation")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_inspect()
{
    last_command="specular_bundle_inspect"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_list()
{
    last_command="specular_bundle_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dir=")
    two_word_flags+=("--dir")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--dir")
    local_nonpersistent_flags+=("--dir=")
    local_nonpersistent_flags+=("-d")
    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_pull()
{
    last_command="specular_bundle_pull"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--insecure")
    local_nonpersistent_flags+=("--insecure")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--user-agent=")
    two_word_flags+=("--user-agent")
    local_nonpersistent_flags+=("--user-agent")
    local_nonpersistent_flags+=("--user-agent=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle_push()
{
    last_command="specular_bundle_push"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--insecure")
    local_nonpersistent_flags+=("--insecure")
    flags+=("--platform=")
    two_word_flags+=("--platform")
    local_nonpersistent_flags+=("--platform")
    local_nonpersistent_flags+=("--platform=")
    flags+=("--user-agent=")
    two_word_flags+=("--user-agent")
    local_nonpersistent_flags+=("--user-agent")
    local_nonpersistent_flags+=("--user-agent=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_bundle()
{
    last_command="specular_bundle"

    command_aliases=()

    commands=()
    commands+=("apply")
    commands+=("approval-status")
    commands+=("approve")
    commands+=("create")
    commands+=("diff")
    commands+=("gate")
    commands+=("inspect")
    commands+=("list")
    commands+=("pull")
    commands+=("push")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_checkpoint_list()
{
    last_command="specular_checkpoint_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_checkpoint_show()
{
    last_command="specular_checkpoint_show"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--verbose")
    flags+=("-v")
    local_nonpersistent_flags+=("--verbose")
    local_nonpersistent_flags+=("-v")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_checkpoint()
{
    last_command="specular_checkpoint"

    command_aliases=()

    commands=()
    commands+=("list")
    commands+=("show")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_completion()
{
    last_command="specular_completion"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    must_have_one_noun+=("bash")
    must_have_one_noun+=("fish")
    must_have_one_noun+=("powershell")
    must_have_one_noun+=("zsh")
    noun_aliases=()
}

_specular_config_edit()
{
    last_command="specular_config_edit"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_config_get()
{
    last_command="specular_config_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_config_path()
{
    last_command="specular_config_path"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_config_set()
{
    last_command="specular_config_set"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_config_view()
{
    last_command="specular_config_view"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_config()
{
    last_command="specular_config"

    command_aliases=()

    commands=()
    commands+=("edit")
    commands+=("get")
    commands+=("path")
    commands+=("set")
    commands+=("view")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug_context()
{
    last_command="specular_debug_context"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug_doctor()
{
    last_command="specular_debug_doctor"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug_explain()
{
    last_command="specular_debug_explain"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--format=")
    two_word_flags+=("--format")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    local_nonpersistent_flags+=("-f")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--output")
    local_nonpersistent_flags+=("--output=")
    local_nonpersistent_flags+=("-o")
    flags+=("--explain")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug_logs_list()
{
    last_command="specular_debug_logs_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug_logs()
{
    last_command="specular_debug_logs"

    command_aliases=()

    commands=()
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--follow")
    flags+=("-f")
    local_nonpersistent_flags+=("--follow")
    local_nonpersistent_flags+=("-f")
    flags+=("--lines=")
    two_word_flags+=("--lines")
    two_word_flags+=("-n")
    local_nonpersistent_flags+=("--lines")
    local_nonpersistent_flags+=("--lines=")
    local_nonpersistent_flags+=("-n")
    flags+=("--tail")
    local_nonpersistent_flags+=("--tail")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    local_nonpersistent_flags+=("--trace")
    local_nonpersistent_flags+=("--trace=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug_status()
{
    last_command="specular_debug_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_debug()
{
    last_command="specular_debug"

    command_aliases=()

    commands=()
    commands+=("context")
    commands+=("doctor")
    commands+=("explain")
    commands+=("logs")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_drift_approve()
{
    last_command="specular_drift_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--message=")
    two_word_flags+=("--message")
    local_nonpersistent_flags+=("--message")
    local_nonpersistent_flags+=("--message=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_drift_check()
{
    last_command="specular_drift_check"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_drift()
{
    last_command="specular_drift"

    command_aliases=()

    commands=()
    commands+=("approve")
    commands+=("check")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_eval_drift()
{
    last_command="specular_eval_drift"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--api-spec=")
    two_word_flags+=("--api-spec")
    local_nonpersistent_flags+=("--api-spec")
    local_nonpersistent_flags+=("--api-spec=")
    flags+=("--checkpoint-dir=")
    two_word_flags+=("--checkpoint-dir")
    local_nonpersistent_flags+=("--checkpoint-dir")
    local_nonpersistent_flags+=("--checkpoint-dir=")
    flags+=("--checkpoint-id=")
    two_word_flags+=("--checkpoint-id")
    local_nonpersistent_flags+=("--checkpoint-id")
    local_nonpersistent_flags+=("--checkpoint-id=")
    flags+=("--fail-on-drift")
    local_nonpersistent_flags+=("--fail-on-drift")
    flags+=("--ignore=")
    two_word_flags+=("--ignore")
    local_nonpersistent_flags+=("--ignore")
    local_nonpersistent_flags+=("--ignore=")
    flags+=("--keep-checkpoint")
    local_nonpersistent_flags+=("--keep-checkpoint")
    flags+=("--lock=")
    two_word_flags+=("--lock")
    local_nonpersistent_flags+=("--lock")
    local_nonpersistent_flags+=("--lock=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--project-root=")
    two_word_flags+=("--project-root")
    local_nonpersistent_flags+=("--project-root")
    local_nonpersistent_flags+=("--project-root=")
    flags+=("--report=")
    two_word_flags+=("--report")
    local_nonpersistent_flags+=("--report")
    local_nonpersistent_flags+=("--report=")
    flags+=("--resume")
    local_nonpersistent_flags+=("--resume")
    flags+=("--spec=")
    two_word_flags+=("--spec")
    local_nonpersistent_flags+=("--spec")
    local_nonpersistent_flags+=("--spec=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_eval_rules()
{
    last_command="specular_eval_rules"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--edit")
    local_nonpersistent_flags+=("--edit")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_eval_run()
{
    last_command="specular_eval_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--scenario=")
    two_word_flags+=("--scenario")
    local_nonpersistent_flags+=("--scenario")
    local_nonpersistent_flags+=("--scenario=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_eval()
{
    last_command="specular_eval"

    command_aliases=()

    commands=()
    commands+=("drift")
    commands+=("rules")
    commands+=("run")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--api-spec=")
    two_word_flags+=("--api-spec")
    local_nonpersistent_flags+=("--api-spec")
    local_nonpersistent_flags+=("--api-spec=")
    flags+=("--checkpoint-dir=")
    two_word_flags+=("--checkpoint-dir")
    local_nonpersistent_flags+=("--checkpoint-dir")
    local_nonpersistent_flags+=("--checkpoint-dir=")
    flags+=("--checkpoint-id=")
    two_word_flags+=("--checkpoint-id")
    local_nonpersistent_flags+=("--checkpoint-id")
    local_nonpersistent_flags+=("--checkpoint-id=")
    flags+=("--fail-on-drift")
    local_nonpersistent_flags+=("--fail-on-drift")
    flags+=("--ignore=")
    two_word_flags+=("--ignore")
    local_nonpersistent_flags+=("--ignore")
    local_nonpersistent_flags+=("--ignore=")
    flags+=("--keep-checkpoint")
    local_nonpersistent_flags+=("--keep-checkpoint")
    flags+=("--lock=")
    two_word_flags+=("--lock")
    local_nonpersistent_flags+=("--lock")
    local_nonpersistent_flags+=("--lock=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--policy=")
    two_word_flags+=("--policy")
    local_nonpersistent_flags+=("--policy")
    local_nonpersistent_flags+=("--policy=")
    flags+=("--project-root=")
    two_word_flags+=("--project-root")
    local_nonpersistent_flags+=("--project-root")
    local_nonpersistent_flags+=("--project-root=")
    flags+=("--report=")
    two_word_flags+=("--report")
    local_nonpersistent_flags+=("--report")
    local_nonpersistent_flags+=("--report=")
    flags+=("--resume")
    local_nonpersistent_flags+=("--resume")
    flags+=("--spec=")
    two_word_flags+=("--spec")
    local_nonpersistent_flags+=("--spec")
    local_nonpersistent_flags+=("--spec=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_generate()
{
    last_command="specular_generate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--complexity=")
    two_word_flags+=("--complexity")
    local_nonpersistent_flags+=("--complexity")
    local_nonpersistent_flags+=("--complexity=")
    flags+=("--max-tokens=")
    two_word_flags+=("--max-tokens")
    local_nonpersistent_flags+=("--max-tokens")
    local_nonpersistent_flags+=("--max-tokens=")
    flags+=("--model-hint=")
    two_word_flags+=("--model-hint")
    local_nonpersistent_flags+=("--model-hint")
    local_nonpersistent_flags+=("--model-hint=")
    flags+=("--priority=")
    two_word_flags+=("--priority")
    local_nonpersistent_flags+=("--priority")
    local_nonpersistent_flags+=("--priority=")
    flags+=("--provider-config=")
    two_word_flags+=("--provider-config")
    local_nonpersistent_flags+=("--provider-config")
    local_nonpersistent_flags+=("--provider-config=")
    flags+=("--router-config=")
    two_word_flags+=("--router-config")
    local_nonpersistent_flags+=("--router-config")
    local_nonpersistent_flags+=("--router-config=")
    flags+=("--stream")
    local_nonpersistent_flags+=("--stream")
    flags+=("--system=")
    two_word_flags+=("--system")
    local_nonpersistent_flags+=("--system")
    local_nonpersistent_flags+=("--system=")
    flags+=("--temperature=")
    two_word_flags+=("--temperature")
    local_nonpersistent_flags+=("--temperature")
    local_nonpersistent_flags+=("--temperature=")
    flags+=("--verbose")
    local_nonpersistent_flags+=("--verbose")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_governance_doctor()
{
    last_command="specular_governance_doctor"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_governance_init()
{
    last_command="specular_governance_init"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    local_nonpersistent_flags+=("--force")
    flags+=("--path=")
    two_word_flags+=("--path")
    local_nonpersistent_flags+=("--path")
    local_nonpersistent_flags+=("--path=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_governance_status()
{
    last_command="specular_governance_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_governance()
{
    last_command="specular_governance"

    command_aliases=()

    commands=()
    commands+=("doctor")
    commands+=("init")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_help()
{
    last_command="specular_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_specular_init()
{
    last_command="specular_init"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cloud")
    local_nonpersistent_flags+=("--cloud")
    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--governance=")
    two_word_flags+=("--governance")
    local_nonpersistent_flags+=("--governance")
    local_nonpersistent_flags+=("--governance=")
    flags+=("--local")
    local_nonpersistent_flags+=("--local")
    flags+=("--mcp=")
    two_word_flags+=("--mcp")
    local_nonpersistent_flags+=("--mcp")
    local_nonpersistent_flags+=("--mcp=")
    flags+=("--no-detect")
    local_nonpersistent_flags+=("--no-detect")
    flags+=("--provider-setup")
    local_nonpersistent_flags+=("--provider-setup")
    flags+=("--providers=")
    two_word_flags+=("--providers")
    local_nonpersistent_flags+=("--providers")
    local_nonpersistent_flags+=("--providers=")
    flags+=("--template=")
    two_word_flags+=("--template")
    local_nonpersistent_flags+=("--template")
    local_nonpersistent_flags+=("--template=")
    flags+=("--yes")
    local_nonpersistent_flags+=("--yes")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plan_create()
{
    last_command="specular_plan_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--estimate")
    local_nonpersistent_flags+=("--estimate")
    flags+=("--feature=")
    two_word_flags+=("--feature")
    local_nonpersistent_flags+=("--feature")
    local_nonpersistent_flags+=("--feature=")
    flags+=("--in=")
    two_word_flags+=("--in")
    two_word_flags+=("-i")
    local_nonpersistent_flags+=("--in")
    local_nonpersistent_flags+=("--in=")
    local_nonpersistent_flags+=("-i")
    flags+=("--lock=")
    two_word_flags+=("--lock")
    local_nonpersistent_flags+=("--lock")
    local_nonpersistent_flags+=("--lock=")
    flags+=("--out=")
    two_word_flags+=("--out")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--out")
    local_nonpersistent_flags+=("--out=")
    local_nonpersistent_flags+=("-o")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plan_explain()
{
    last_command="specular_plan_explain"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plan_review()
{
    last_command="specular_plan_review"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plan_validate()
{
    last_command="specular_plan_validate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plan_visualize()
{
    last_command="specular_plan_visualize"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plan()
{
    last_command="specular_plan"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("explain")
    commands+=("review")
    commands+=("validate")
    commands+=("visualize")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--estimate")
    local_nonpersistent_flags+=("--estimate")
    flags+=("--feature=")
    two_word_flags+=("--feature")
    local_nonpersistent_flags+=("--feature")
    local_nonpersistent_flags+=("--feature=")
    flags+=("--in=")
    two_word_flags+=("--in")
    two_word_flags+=("-i")
    local_nonpersistent_flags+=("--in")
    local_nonpersistent_flags+=("--in=")
    local_nonpersistent_flags+=("-i")
    flags+=("--lock=")
    two_word_flags+=("--lock")
    local_nonpersistent_flags+=("--lock")
    local_nonpersistent_flags+=("--lock=")
    flags+=("--out=")
    two_word_flags+=("--out")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--out")
    local_nonpersistent_flags+=("--out=")
    local_nonpersistent_flags+=("-o")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_project_create()
{
    last_command="specular_platform_project_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--visibility=")
    two_word_flags+=("--visibility")
    local_nonpersistent_flags+=("--visibility")
    local_nonpersistent_flags+=("--visibility=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_project_delete()
{
    last_command="specular_platform_project_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    local_nonpersistent_flags+=("--force")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_project_list()
{
    last_command="specular_platform_project_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--page=")
    two_word_flags+=("--page")
    local_nonpersistent_flags+=("--page")
    local_nonpersistent_flags+=("--page=")
    flags+=("--page-size=")
    two_word_flags+=("--page-size")
    local_nonpersistent_flags+=("--page-size")
    local_nonpersistent_flags+=("--page-size=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_project_show()
{
    last_command="specular_platform_project_show"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_project()
{
    last_command="specular_platform_project"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("list")
    commands+=("show")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_session_create()
{
    last_command="specular_platform_session_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--title=")
    two_word_flags+=("--title")
    local_nonpersistent_flags+=("--title")
    local_nonpersistent_flags+=("--title=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_session_list()
{
    last_command="specular_platform_session_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--page=")
    two_word_flags+=("--page")
    local_nonpersistent_flags+=("--page")
    local_nonpersistent_flags+=("--page=")
    flags+=("--page-size=")
    two_word_flags+=("--page-size")
    local_nonpersistent_flags+=("--page-size")
    local_nonpersistent_flags+=("--page-size=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_session_messages()
{
    last_command="specular_platform_session_messages"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--page=")
    two_word_flags+=("--page")
    local_nonpersistent_flags+=("--page")
    local_nonpersistent_flags+=("--page=")
    flags+=("--page-size=")
    two_word_flags+=("--page-size")
    local_nonpersistent_flags+=("--page-size")
    local_nonpersistent_flags+=("--page-size=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_session_send()
{
    last_command="specular_platform_session_send"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_session_show()
{
    last_command="specular_platform_session_show"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform_session()
{
    last_command="specular_platform_session"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("list")
    commands+=("messages")
    commands+=("send")
    commands+=("show")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_platform()
{
    last_command="specular_platform"

    command_aliases=()

    commands=()
    commands+=("project")
    commands+=("session")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_disable()
{
    last_command="specular_plugin_disable"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_enable()
{
    last_command="specular_plugin_enable"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_health()
{
    last_command="specular_plugin_health"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_info()
{
    last_command="specular_plugin_info"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_install()
{
    last_command="specular_plugin_install"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_list()
{
    last_command="specular_plugin_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin_uninstall()
{
    last_command="specular_plugin_uninstall"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    local_nonpersistent_flags+=("--force")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_plugin()
{
    last_command="specular_plugin"

    command_aliases=()

    commands=()
    commands+=("disable")
    commands+=("enable")
    commands+=("health")
    commands+=("info")
    commands+=("install")
    commands+=("list")
    commands+=("uninstall")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("remove")
        aliashash["remove"]="uninstall"
        command_aliases+=("rm")
        aliashash["rm"]="uninstall"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_policy_approve()
{
    last_command="specular_policy_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--message=")
    two_word_flags+=("--message")
    local_nonpersistent_flags+=("--message")
    local_nonpersistent_flags+=("--message=")
    flags+=("--user=")
    two_word_flags+=("--user")
    local_nonpersistent_flags+=("--user")
    local_nonpersistent_flags+=("--user=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_policy_diff()
{
    last_command="specular_policy_diff"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--unified")
    local_nonpersistent_flags+=("--unified")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_policy_init()
{
    last_command="specular_policy_init"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--path=")
    two_word_flags+=("--path")
    local_nonpersistent_flags+=("--path")
    local_nonpersistent_flags+=("--path=")
    flags+=("--template=")
    two_word_flags+=("--template")
    local_nonpersistent_flags+=("--template")
    local_nonpersistent_flags+=("--template=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_policy_list()
{
    last_command="specular_policy_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_policy_validate()
{
    last_command="specular_policy_validate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--strict")
    local_nonpersistent_flags+=("--strict")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_policy()
{
    last_command="specular_policy"

    command_aliases=()

    commands=()
    commands+=("approve")
    commands+=("diff")
    commands+=("init")
    commands+=("list")
    commands+=("validate")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_prewarm()
{
    last_command="specular_prewarm"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--cache-dir=")
    two_word_flags+=("--cache-dir")
    local_nonpersistent_flags+=("--cache-dir")
    local_nonpersistent_flags+=("--cache-dir=")
    flags+=("--concurrency=")
    two_word_flags+=("--concurrency")
    local_nonpersistent_flags+=("--concurrency")
    local_nonpersistent_flags+=("--concurrency=")
    flags+=("--export=")
    two_word_flags+=("--export")
    local_nonpersistent_flags+=("--export")
    local_nonpersistent_flags+=("--export=")
    flags+=("--import=")
    two_word_flags+=("--import")
    local_nonpersistent_flags+=("--import")
    local_nonpersistent_flags+=("--import=")
    flags+=("--max-age=")
    two_word_flags+=("--max-age")
    local_nonpersistent_flags+=("--max-age")
    local_nonpersistent_flags+=("--max-age=")
    flags+=("--plan=")
    two_word_flags+=("--plan")
    local_nonpersistent_flags+=("--plan")
    local_nonpersistent_flags+=("--plan=")
    flags+=("--prune")
    local_nonpersistent_flags+=("--prune")
    flags+=("--verbose")
    local_nonpersistent_flags+=("--verbose")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_provider_add()
{
    last_command="specular_provider_add"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_provider_doctor()
{
    last_command="specular_provider_doctor"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_provider_init()
{
    last_command="specular_provider_init"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    local_nonpersistent_flags+=("--force")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_provider_list()
{
    last_command="specular_provider_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_provider_remove()
{
    last_command="specular_provider_remove"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_provider()
{
    last_command="specular_provider"

    command_aliases=()

    commands=()
    commands+=("add")
    commands+=("doctor")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("health")
        aliashash["health"]="doctor"
    fi
    commands+=("init")
    commands+=("list")
    commands+=("remove")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="remove"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_route_explain()
{
    last_command="specular_route_explain"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_route_list()
{
    last_command="specular_route_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--available")
    local_nonpersistent_flags+=("--available")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_route_override()
{
    last_command="specular_route_override"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_route()
{
    last_command="specular_route"

    command_aliases=()

    commands=()
    commands+=("explain")
    commands+=("list")
    commands+=("override")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_session_list()
{
    last_command="specular_session_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_session_show()
{
    last_command="specular_session_show"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--verbose")
    flags+=("-v")
    local_nonpersistent_flags+=("--verbose")
    local_nonpersistent_flags+=("-v")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_session()
{
    last_command="specular_session"

    command_aliases=()

    commands=()
    commands+=("list")
    commands+=("show")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_approve()
{
    last_command="specular_spec_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_diff()
{
    last_command="specular_spec_diff"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_edit()
{
    last_command="specular_spec_edit"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_generate()
{
    last_command="specular_spec_generate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--in=")
    two_word_flags+=("--in")
    two_word_flags+=("-i")
    local_nonpersistent_flags+=("--in")
    local_nonpersistent_flags+=("--in=")
    local_nonpersistent_flags+=("-i")
    flags+=("--out=")
    two_word_flags+=("--out")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--out")
    local_nonpersistent_flags+=("--out=")
    local_nonpersistent_flags+=("-o")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_lock()
{
    last_command="specular_spec_lock"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--in=")
    two_word_flags+=("--in")
    two_word_flags+=("-i")
    local_nonpersistent_flags+=("--in")
    local_nonpersistent_flags+=("--in=")
    local_nonpersistent_flags+=("-i")
    flags+=("--note=")
    two_word_flags+=("--note")
    local_nonpersistent_flags+=("--note")
    local_nonpersistent_flags+=("--note=")
    flags+=("--out=")
    two_word_flags+=("--out")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--out")
    local_nonpersistent_flags+=("--out=")
    local_nonpersistent_flags+=("-o")
    flags+=("--version=")
    two_word_flags+=("--version")
    local_nonpersistent_flags+=("--version")
    local_nonpersistent_flags+=("--version=")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_new()
{
    last_command="specular_spec_new"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from=")
    two_word_flags+=("--from")
    local_nonpersistent_flags+=("--from")
    local_nonpersistent_flags+=("--from=")
    flags+=("--list")
    local_nonpersistent_flags+=("--list")
    flags+=("--out=")
    two_word_flags+=("--out")
    two_word_flags+=("-o")
    local_nonpersistent_flags+=("--out")
    local_nonpersistent_flags+=("--out=")
    local_nonpersistent_flags+=("-o")
    flags+=("--preset=")
    two_word_flags+=("--preset")
    local_nonpersistent_flags+=("--preset")
    local_nonpersistent_flags+=("--preset=")
    flags+=("--strict")
    local_nonpersistent_flags+=("--strict")
    flags+=("--tui")
    local_nonpersistent_flags+=("--tui")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec_validate()
{
    last_command="specular_spec_validate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--in=")
    two_word_flags+=("--in")
    two_word_flags+=("-i")
    local_nonpersistent_flags+=("--in")
    local_nonpersistent_flags+=("--in=")
    local_nonpersistent_flags+=("-i")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_spec()
{
    last_command="specular_spec"

    command_aliases=()

    commands=()
    commands+=("approve")
    commands+=("diff")
    commands+=("edit")
    commands+=("generate")
    commands+=("lock")
    commands+=("new")
    commands+=("validate")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_version()
{
    last_command="specular_version"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--json")
    local_nonpersistent_flags+=("--json")
    flags+=("--verbose")
    flags+=("-v")
    local_nonpersistent_flags+=("--verbose")
    local_nonpersistent_flags+=("-v")
    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_specular_root_command()
{
    last_command="specular"

    command_aliases=()

    commands=()
    commands+=("approvals")
    commands+=("approve")
    commands+=("auth")
    commands+=("auto")
    commands+=("build")
    commands+=("bundle")
    commands+=("checkpoint")
    commands+=("completion")
    commands+=("config")
    commands+=("debug")
    commands+=("drift")
    commands+=("eval")
    commands+=("generate")
    commands+=("governance")
    commands+=("help")
    commands+=("init")
    commands+=("plan")
    commands+=("platform")
    commands+=("plugin")
    commands+=("policy")
    commands+=("prewarm")
    commands+=("provider")
    commands+=("route")
    commands+=("session")
    commands+=("spec")
    commands+=("version")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--explain")
    flags+=("--format=")
    two_word_flags+=("--format")
    flags+=("--home=")
    two_word_flags+=("--home")
    flags+=("--log-level=")
    two_word_flags+=("--log-level")
    flags+=("--no-color")
    flags+=("--quiet")
    flags+=("-q")
    flags+=("--trace=")
    two_word_flags+=("--trace")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

__start_specular()
{
    local cur prev words cword split
    declare -A flaghash 2>/dev/null || :
    declare -A aliashash 2>/dev/null || :
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -s || return
    else
        __specular_init_completion -n "=" || return
    fi

    local c=0
    local flag_parsing_disabled=
    local flags=()
    local two_word_flags=()
    local local_nonpersistent_flags=()
    local flags_with_completion=()
    local flags_completion=()
    local commands=("specular")
    local command_aliases=()
    local must_have_one_flag=()
    local must_have_one_noun=()
    local has_completion_function=""
    local last_command=""
    local nouns=()
    local noun_aliases=()

    __specular_handle_word
}

if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_specular specular
else
    complete -o default -o nospace -F __start_specular specular
fi

# ex: ts=4 sw=4 et filetype=sh

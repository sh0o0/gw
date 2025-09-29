# Function definitions start here
function __gw_get_symlink_patterns
    echo "**/.vscode/*"
    echo "**/.claude/*"
    echo "**/.env*"
    echo "**/.github/prompts/*.local.prompt.md"
    echo "**/.ignored/**"
    echo "**/.serena/**"
end

function __gw_get_exclude_patterns
    echo "**/node_modules/*"
end

# ---- helpers: messaging / prerequisites ------------------------------------
function __gw_err
    echo (string join " " $argv) >&2
end

function __gw_warn
    echo "Warning:" (string join " " $argv) >&2
end

function __gw_info
    echo (string join " " $argv)
end

function __gw_require_cmd
    set -l cmd $argv[1]
    if not command -q $cmd
        __gw_err "Required command not found:" $cmd
        return 1
    end
end

function gw
    set -l cmd $argv[1]

    switch $cmd
        case link
            if test (count $argv) -ge 2
                __gw_link $argv[2]
            else
                echo "Error: Path required for link command"
                return 1
            end
        case unlink
            if test (count $argv) -ge 2
                __gw_unlink $argv[2]
            else
                echo "Error: Path required for unlink command"
                return 1
            end
        case switch
            if test (count $argv) -ge 2
                __gw_switch_to_branch $argv[2]
            else
                __gw_switch_worktree
            end
        case checkout
            if test (count $argv) -ge 2
                __gw_checkout_branch $argv[2]
            else
                echo "Error: Branch name required for checkout command"
                return 1
            end
        case restore
            if test (count $argv) -ge 2
                __gw_restore_branch $argv[2]
            else
                echo "Error: Branch name required for restore command"
                return 1
            end
        case list
            git worktree list
        case prune
            git worktree prune
        case remove
            if test (count $argv) -ge 2
                set -l force_flag false
                set -l branch_args

                # Check for --force flag and collect branch names
                for arg in $argv[2..-1]
                    if test "$arg" = "--force"
                        set force_flag true
                    else
                        set -a branch_args "$arg"
                    end
                end

                if test (count $branch_args) -gt 0
                    __gw_remove_multiple_worktrees $force_flag $branch_args
                else
                    __gw_remove_worktree $force_flag
                end
            else
                __gw_remove_worktree false
            end
        case '*'
            if test -n "$cmd"
                __gw_err "Unknown command:" $cmd
            end
            __gw_show_help
            return 1
    end
end

function __gw_show_help
    __gw_info "Available commands:"
    __gw_info "  link <path>              Move file to base worktree and create symlink"
    __gw_info "  unlink <path>            Replace symlink with a real file/dir by copying its target"
    __gw_info "  switch                    Fuzzy search and cd to worktree"
    __gw_info "  switch <branch_name>      Switch directly to worktree for specified branch"
    __gw_info "  checkout <branch_name>    Switch to existing worktree or create new one for branch"
    __gw_info "  restore <branch_name>     Create new worktree for specified branch"
    __gw_info "  list                      List all worktrees"
    __gw_info "  prune                     Prune worktrees"
    __gw_info "  remove                    Fuzzy search and remove worktree"
    __gw_info "  remove --force            Fuzzy search and force remove worktree"
    __gw_info "  remove <branch_name>...   Remove worktrees for specified branch(es)"
    __gw_info "  remove --force <branch_name>...   Force remove worktrees for specified branch(es)"
end

function __gw_checkout_branch
    set -l branch_name $argv[1]

    # Capture current state before switching
    set -l prev_rev (git rev-parse --verify HEAD 2>/dev/null)
    set -l prev_branch (__gw_get_branch_from_worktree (__gw_get_current_worktree_path))

    # Try to find existing worktree for the branch
    set -l worktree_path (__gw_find_worktree_by_branch "$branch_name")

    if test $status -eq 0 -a -n "$worktree_path"
        # Worktree exists, switch to it
        __gw_switch_to_path "$worktree_path"
    else
        # Worktree doesn't exist, create it
        __gw_create_worktree "$branch_name"
    end

    # After switching/creation, we're now in the target worktree; capture new state
    set -l new_rev (git rev-parse --verify HEAD 2>/dev/null)
    set -l new_branch (__gw_get_branch_from_worktree (__gw_get_current_worktree_path))

    # Run post-checkout hook if available (mimic git's 3 args: oldrev newrev isBranchCheckout)
    __gw_run_post_checkout "$prev_rev" "$new_rev" 1 "$prev_branch" "$new_branch"
end

function __gw_unlink
    set -l input_path $argv[1]

    if test -z "$input_path"
        __gw_err "Error: Path required"
        return 1
    end

    set -l source_abs (__gw_resolve_absolute_path "$input_path")
    or begin
        __gw_err "Error: Failed to resolve path:" $input_path
        return 1
    end

    if not test -e "$source_abs"
        __gw_err "Error: Path not found:" $source_abs
        return 1
    end

    if not test -L "$source_abs"
        __gw_err "Error: Not a symlink:" $source_abs
        return 1
    end

    set -l target (readlink "$source_abs")
    set -l link_dir (dirname "$source_abs")
    set -l base_name (basename "$source_abs")

    # If the symlink resolves to a directory, copy recursively; otherwise copy as a file
    if test -d "$source_abs"
        set -l tmp_dir "$link_dir/.gw_unlink_tmp_dir.$RANDOM"
        mkdir -p "$tmp_dir"
        or begin
            __gw_err "Error: Failed to create temp directory:" $tmp_dir
            return 1
        end

        # Copy the referent (follow links) into the temp dir
        cp -pRL "$source_abs" "$tmp_dir/"
        or begin
            __gw_err "Error: Failed to copy target directory of symlink"
            command rm -rf "$tmp_dir" >/dev/null 2>&1
            return 1
        end

        # Replace symlink with the copied directory
        rm "$source_abs"
        or begin
            __gw_err "Error: Failed to remove symlink:" $source_abs
            command rm -rf "$tmp_dir" >/dev/null 2>&1
            return 1
        end

        mv "$tmp_dir/$base_name" "$source_abs"
        or begin
            __gw_err "Error: Failed to materialize directory at:" $source_abs
            command rm -rf "$tmp_dir" >/dev/null 2>&1
            return 1
        end

        command rmdir "$tmp_dir" >/dev/null 2>&1
    else
        set -l tmp_file "$source_abs.__gw_unlink_tmp__.$RANDOM"

        cp -pL "$source_abs" "$tmp_file"
        or begin
            __gw_err "Error: Failed to copy target file of symlink"
            return 1
        end

        rm "$source_abs"
        or begin
            __gw_err "Error: Failed to remove symlink:" $source_abs
            command rm -f "$tmp_file" >/dev/null 2>&1
            return 1
        end

        mv "$tmp_file" "$source_abs"
        or begin
            __gw_err "Error: Failed to materialize file at:" $source_abs
            command rm -f "$tmp_file" >/dev/null 2>&1
            return 1
        end
    end

    if test -n "$target"
        __gw_info "Unlinked: $source_abs (copied from: $target)"
    else
        __gw_info "Unlinked: $source_abs"
    end
end

function __gw_create_single_symlink
    set -l source_item $argv[1]
    set -l target_item $argv[2]
    set -l gitignored_file $argv[3]

    if not test -e "$source_item"
        return 1
    end

    set -l target_dir (dirname "$target_item")
    if not test -d "$target_dir"
        mkdir -p "$target_dir"
    end

    if test -e "$target_item"
        rm -rf "$target_item"
    end

    ln -s "$source_item" "$target_item"
    __gw_info "Created symlink: /$gitignored_file"
    return 0
end

function __gw_create_symlinks
    set -l source_path $argv[1]
    set -l target_path $argv[2]

    set -l gitignored_files (git -C "$source_path" ls-files --others -i --exclude-standard)
    if test -z "$gitignored_files"
        __gw_info "No gitignored files found to symlink"
        return 0
    end

    set -l patterns (__gw_get_symlink_patterns)
    set -l exclude_patterns (__gw_get_exclude_patterns)
    set -l linked_count 0

    for gitignored_file in $gitignored_files
        # Check if file should be excluded
        set -l should_exclude false
        for exclude_pattern in $exclude_patterns
            if string match -q "$exclude_pattern" "$gitignored_file"
                set should_exclude true
                break
            end
        end

        if test "$should_exclude" = true
            continue
        end

        for pattern in $patterns
            if string match -q "$pattern" "/$gitignored_file"
                set -l source_item "$source_path/$gitignored_file"
                set -l target_item "$target_path/$gitignored_file"

                if __gw_create_single_symlink "$source_item" "$target_item" "$gitignored_file"
                    set linked_count (math $linked_count + 1)
                end
                break
            end
        end
    end

    if test $linked_count -eq 0
        __gw_info "No matching gitignored files found to symlink"
    else
        __gw_info "Created $linked_count symlinks to worktree"
    end
end

function __gw_validate_git_repository
    if not git rev-parse --git-dir >/dev/null 2>&1
        __gw_err "Error: Not in a git repository"
        return 1
    end
    return 0
end

function __gw_get_git_root
    if not __gw_validate_git_repository
        return 1
    end
    git rev-parse --show-toplevel
end

function __gw_parse_remote_url
    set -l remote_url (git remote get-url origin 2>/dev/null)
    if test -z "$remote_url"
        # Return empty to indicate no remote, don't error
        return 1
    end

    # Parse different URL formats
    # SSH format: git@github.com:user/repo.git
    # HTTPS format: https://github.com/user/repo.git
    # Git format: git://github.com/user/repo.git

    set -l domain ""
    set -l org ""
    set -l repo ""

    if string match -q "git@*" "$remote_url"
        # SSH format: git@domain:org/repo.git
        set -l parts (string split "@" "$remote_url")
        set -l host_path (string split ":" "$parts[2]")
        set domain "$host_path[1]"
        set -l path_parts (string split "/" "$host_path[2]")
        set org "$path_parts[1]"
        set repo (string replace ".git" "" "$path_parts[2]")
    else if string match -q "https://*" "$remote_url"
        # HTTPS format: https://domain/org/repo.git
        set -l url_without_protocol (string replace -r "^https://" "" "$remote_url")
        set -l parts (string split "/" "$url_without_protocol")
        set domain "$parts[1]"
        set org "$parts[2]"
        set repo (string replace ".git" "" "$parts[3]")
    else if string match -q "http://*" "$remote_url"
        # HTTP format: http://domain/org/repo.git
        set -l url_without_protocol (string replace -r "^http://" "" "$remote_url")
        set -l parts (string split "/" "$url_without_protocol")
        set domain "$parts[1]"
        set org "$parts[2]"
        set repo (string replace ".git" "" "$parts[3]")
    else if string match -q "git://*" "$remote_url"
        # Git format: git://domain/org/repo.git
        set -l url_without_protocol (string replace "git://" "" "$remote_url")
        set -l parts (string split "/" "$url_without_protocol")
        set domain "$parts[1]"
        set org "$parts[2]"
        set repo (string replace ".git" "" "$parts[3]")
    else
        echo "Error: Unsupported remote URL format: $remote_url" >&2
        return 1
    end

    if test -z "$domain" -o -z "$org" -o -z "$repo"
        echo "Error: Could not parse remote URL: $remote_url" >&2
        return 1
    end

    echo "$domain"
    echo "$org"
    echo "$repo"
end

function __gw_get_worktree_base_path
    set -l git_root (__gw_get_git_root)
    or return 1

    if test -z "$git_root"
        __gw_err "Error: Git root is empty"
        return 1
    end

    set -l remote_info (__gw_parse_remote_url)
    if test $status -eq 0
        # Remote exists, use remote-based path
        set -l remote_info_lines (echo "$remote_info" | string split " ")
        set -l domain "$remote_info_lines[1]"
        set -l org "$remote_info_lines[2]"
        set -l repo "$remote_info_lines[3]"

        if test -z "$domain" -o -z "$org" -o -z "$repo"
            __gw_err "Error: Could not parse remote information"
            return 1
        end

        echo "$HOME/.worktrees/$domain/$org/$repo"
    else
        # No remote, use local path based on git root relative to HOME
        set -l relative_path (string replace "$HOME/" "" "$git_root")
        echo "$HOME/.worktrees/local/$relative_path"
    end
end

# ---- hooks ------------------------------------------------------------------
function __gw_get_hooks_dir
    # Prefer the original/primary worktree (actual project root),
    # fall back to current repo root if not found.
    set -l primary_root (__gw_get_primary_worktree_path)
    if test $status -eq 0 -a -n "$primary_root"
        echo "$primary_root/.gw/hooks"
        return 0
    end

    set -l git_root (__gw_get_git_root)
    or return 1
    echo "$git_root/.gw/hooks"
end

function __gw_run_hook
    # in: hook_name, args...
    set -l hook_name $argv[1]
    set -l args $argv[2..-1]

    set -l hooks_dir (__gw_get_hooks_dir)
    or return 1

    if not test -d "$hooks_dir"
        return 1
    end

    set -l hook_file "$hooks_dir/$hook_name"
    set -l hook_dir "$hooks_dir/$hook_name.d"
    set -l ran false
    set -l all_ok true

    if test -f "$hook_file"
        if test -x "$hook_file"
            $hook_file $args
            or begin
                __gw_warn "Hook failed:" $hook_file
                set all_ok false
            end
            set ran true
        else
            __gw_warn "Hook exists but not executable:" $hook_file
        end
    end

    if test -d "$hook_dir"
        for f in (ls -A "$hook_dir" 2>/dev/null | sort)
            set -l path "$hook_dir/$f"
            if test -f "$path" -a -x "$path"
                $path $args
                or begin
                    __gw_warn "Hook failed:" $path
                    set all_ok false
                end
                set ran true
            end
        end
    end

    if test "$ran" = false
        return 1
    end

    if test "$all_ok" = true
        return 0
    else
        return 2
    end
end

function __gw_run_post_checkout
    # args mimic git post-checkout: oldrev newrev isBranchCheckout
    # extra: prev_branch new_branch for convenience via env vars
    set -l oldrev $argv[1]
    set -l newrev $argv[2]
    set -l is_branch $argv[3]
    set -l prev_branch $argv[4]
    set -l new_branch $argv[5]

    set -l hooks_dir (__gw_get_hooks_dir)
    or return 0

    if not test -d "$hooks_dir"
        return 0
    end

    set -lx GW_HOOK_NAME "post-checkout"
    set -lx GW_PREV_BRANCH "$prev_branch"
    set -lx GW_NEW_BRANCH "$new_branch"

    __gw_run_hook "post-checkout" "$oldrev" "$newrev" "$is_branch"
    set -l hook_status $status

    # Clean up exported env vars
    set -e GW_HOOK_NAME
    set -e GW_PREV_BRANCH
    set -e GW_NEW_BRANCH

    switch $hook_status
        case 0
            __gw_info "post-checkout hook executed"
        case 2
            __gw_warn "post-checkout hook completed with errors"
        case '*'
            # No hook found -> silent
    end
end

# ---- helpers: worktree path computation / post steps ------------------------
function __gw_compute_worktree_path
    # in: branch_name
    set -l branch_name $argv[1]

    if test -z "$branch_name"
        __gw_err "Error: Branch name required"
        return 1
    end

    set -l worktree_base_path (__gw_get_worktree_base_path)
    or return 1

    if test -z "$worktree_base_path"
        __gw_err "Error: Could not determine worktree base path"
        return 1
    end

    if not test -d "$worktree_base_path"
        mkdir -p "$worktree_base_path"
        or begin
            __gw_err "Error: Failed to create worktree base directory:" $worktree_base_path
            return 1
        end
    end

    set -l parsed_branch_name (string replace -a "/" "-" "$branch_name")
    set -l worktree_path "$worktree_base_path/$parsed_branch_name"

    if test -z "$worktree_path" -o "$worktree_path" = "/"
        __gw_err "Error: Invalid worktree path:" $worktree_path
        return 1
    end

    echo "$worktree_path"
end

function __gw_post_create_worktree
    # in: worktree_path, relative_path
    set -l worktree_path $argv[1]
    set -l relative_path $argv[2]

    set -l git_root (__gw_get_git_root)
    __gw_create_symlinks "$git_root" "$worktree_path"
    __gw_navigate_to_relative_path "$worktree_path" "$relative_path"
end

function __gw_parse_single_worktree
    set -l lines $argv
    set -l path ""
    set -l branch ""

    for line in $lines
        switch $line
            case "worktree *"
                set path (string replace "worktree " "" "$line")
            case "branch *"
                set branch (string replace "branch refs/heads/" "" "$line")
            case "HEAD *"
                set branch HEAD
        end
    end

    if test -n "$path"
        printf "%s\t%s\n" "$path" "$branch"
    end
end

function __gw_format_worktree_entry
    set -l path $argv[1]
    set -l branch $argv[2]

    if test -z "$branch"
        set branch "(detached)"
    end

    echo "[$branch]  $path"
end

function __gw_parse_worktree_info
    set -l worktree_info $argv
    set -l display_list
    set -l path_list
    set -l current_entry

    for line in $worktree_info
        if test "$line" = ""
            if test -n "$current_entry"
                set -l parsed (__gw_parse_single_worktree $current_entry)
                if test -n "$parsed"
                    set -l path (echo "$parsed" | cut -f1)
                    set -l branch (echo "$parsed" | cut -f2)
                    set -a display_list (__gw_format_worktree_entry "$path" "$branch")
                    set -a path_list "$path"
                end
                set current_entry
            end
        else
            set -a current_entry "$line"
        end
    end

    # Handle last entry
    if test -n "$current_entry"
        set -l parsed (__gw_parse_single_worktree $current_entry)
        if test -n "$parsed"
            set -l path (echo "$parsed" | cut -f1)
            set -l branch (echo "$parsed" | cut -f2)
            set -a display_list (__gw_format_worktree_entry "$path" "$branch")
            set -a path_list "$path"
        end
    end

    printf "%s\n" $display_list
    echo ---
    printf "%s\n" $path_list
end

function __gw_resolve_absolute_path
    set -l input $argv[1]
    if test -z "$input"
        return 1
    end

    if string match -q "/*" "$input"
        echo "$input"
        return 0
    end

    set -l prev (pwd)
    set -l dir (dirname "$input")
    set -l base (basename "$input")

    if string match -q "/*" "$dir"
        cd "$dir"
    else
        cd "$prev/$dir"
    end

    set -l abs_dir (pwd -P)
    cd "$prev"
    echo "$abs_dir/$base"
end

function __gw_get_worktree_lists
    set -l worktree_info (git worktree list --porcelain)
    or begin
        __gw_err "Error: No worktrees found"
        return 1
    end

    set -l parsed_output (__gw_parse_worktree_info $worktree_info)
    set -l separator_index

    for i in (seq (count $parsed_output))
        if test "$parsed_output[$i]" = ---
            set separator_index $i
            break
        end
    end

    if test -z "$separator_index"
        __gw_err "Error: Failed to parse worktree info"
        return 1
    end

    set -l display_list $parsed_output[1..(math $separator_index - 1)]
    set -l path_list $parsed_output[(math $separator_index + 1)..-1]

    printf "%s\n" $display_list
    echo ---
    printf "%s\n" $path_list
end

function __gw_get_primary_worktree_path
    # Determine the original repository working tree path based on the common git dir.
    # `git rev-parse --git-common-dir` yields the shared .git directory even in linked worktrees.
    set -l cg (git rev-parse --git-common-dir 2>/dev/null)
    or return 1

    if test -z "$cg"
        return 1
    end

    set -l common_git_dir (__gw_resolve_absolute_path "$cg")
    or return 1

    set -l primary_root (dirname "$common_git_dir")
    if test -d "$primary_root"
        echo "$primary_root"
        return 0
    end

    return 1
end

function __gw_link
    set -l input_path $argv[1]

    if test -z "$input_path"
        __gw_err "Error: Path required"
        return 1
    end

    set -l source_abs (__gw_resolve_absolute_path "$input_path")
    or begin
        __gw_err "Error: Failed to resolve path:" $input_path
        return 1
    end

    if not test -e "$source_abs"
        __gw_err "Error: File not found:" $source_abs
        return 1
    end

    set -l current_root (__gw_get_git_root)
    or return 1

    set -l primary_root (__gw_get_primary_worktree_path)
    or begin
        __gw_err "Error: Failed to locate primary worktree"
        return 1
    end

    if test "$current_root" = "$primary_root"
        __gw_err "Error: You are in the primary worktree; nothing to link"
        return 1
    end

    if not string match -q "$current_root/*" "$source_abs"
        __gw_err "Error: Path must be within current worktree:" $current_root
        return 1
    end

    set -l rel_path (string replace "$current_root/" "" "$source_abs")
    set -l dest_path "$primary_root/$rel_path"
    set -l dest_dir (dirname "$dest_path")

    if not test -d "$dest_dir"
        mkdir -p "$dest_dir"
        or begin
            __gw_err "Error: Failed to create directory:" $dest_dir
            return 1
        end
    end

    if test -e "$dest_path"
        __gw_err "Error: Destination already exists:" $dest_path
        return 1
    end

    mv "$source_abs" "$dest_path"
    or begin
        __gw_err "Error: Failed to move to:" $dest_path
        return 1
    end

    ln -s "$dest_path" "$source_abs"
    or begin
        __gw_err "Error: Failed to create symlink at:" $source_abs
        return 1
    end

    __gw_info "Linked: $source_abs -> $dest_path"
end

function __gw_select_worktree
    set -l prompt_message $argv[1]
    set -l exclude_current $argv[2]

    set -l lists_output (__gw_get_worktree_lists)
    or return 1

    set -l separator_index
    for i in (seq (count $lists_output))
        if test "$lists_output[$i]" = ---
            set separator_index $i
            break
        end
    end

    set -l display_list $lists_output[1..(math $separator_index - 1)]
    set -l path_list $lists_output[(math $separator_index + 1)..-1]

    # Filter out current worktree if requested
    if test "$exclude_current" = true
        set -l current_worktree (__gw_get_current_worktree_path)
        if test -n "$current_worktree"
            set -l filtered_display_list
            set -l filtered_path_list

            for i in (seq (count $path_list))
                if test "$path_list[$i]" != "$current_worktree"
                    set -a filtered_display_list "$display_list[$i]"
                    set -a filtered_path_list "$path_list[$i]"
                end
            end

            set display_list $filtered_display_list
            set path_list $filtered_path_list
        end
    end

    if test (count $display_list) -eq 0
        echo "Error: No worktrees available for selection" >&2
        return 1
    end

    __gw_require_cmd fzf
    or return 1
    set -l selected_item (printf '%s\n' $display_list | fzf --prompt="$prompt_message")
    or return 1

    for i in (seq (count $display_list))
        if test "$display_list[$i]" = "$selected_item"
            echo $path_list[$i]
            return 0
        end
    end

    return 1
end

function __gw_execute_with_selected_worktree
    set -l prompt_message $argv[1]
    set -l action_function $argv[2]
    set -l exclude_current $argv[3]

    set -l selected_path (__gw_select_worktree "$prompt_message" "$exclude_current")
    if test $status -eq 0
        eval $action_function "$selected_path"
    else
        __gw_info "Selection cancelled"
        return 1
    end
end

function __gw_get_branch_from_worktree
    set -l path $argv[1]

    set -l branch_name (git -C "$path" branch --show-current 2>/dev/null)
    if test -n "$branch_name"
        echo "$branch_name"
        return 0
    end

    # Fallback: get branch from git worktree list
    set -l worktree_info (git worktree list --porcelain | grep -A2 "^worktree $path\$")
    for line in $worktree_info
        if string match -q "branch refs/heads/*" "$line"
            echo (string replace "branch refs/heads/" "" "$line")
            return 0
        end
    end

    return 1
end

function __gw_remove_worktree_at_path
    set -l path $argv[1]
    set -l force_flag $argv[2]

    set -l branch_name (__gw_get_branch_from_worktree "$path")

    __gw_info "Removing worktree:" $path
    if test "$force_flag" = true
        git worktree remove --force "$path"
    else
        git worktree remove "$path"
    end
    or begin
        __gw_err "Error: Failed to remove worktree"
        return 1
    end

    if test -n "$branch_name"
        __gw_info "Deleting branch:" $branch_name
        if git branch -D "$branch_name"
            __gw_info "Successfully deleted branch:" $branch_name
        else
            __gw_warn "Failed to delete branch:" $branch_name
        end
    else
        __gw_warn "Could not determine branch name"
    end
end

function __gw_get_relative_path_from_git_root
    set -l git_root (__gw_get_git_root)
    or return 1

    set -l current_dir (pwd)

    if string match -q "$git_root/*" "$current_dir"
        string replace "$git_root/" "" "$current_dir"
    else if test "$current_dir" = "$git_root"
        echo "."
    else
        echo "."
    end
end

function __gw_navigate_to_relative_path
    set -l worktree_path $argv[1]
    set -l relative_path $argv[2]

    if test "$relative_path" = "."
        cd "$worktree_path"
        return
    end

    set -l target_path "$worktree_path/$relative_path"
    if test -d "$target_path"
        cd "$target_path"
    else
        __gw_warn "Directory" $relative_path "does not exist in worktree, staying at root"
        cd "$worktree_path"
    end
end

function __gw_create_worktree_from_current_branch
    set -l branch_name $argv[1]

    # Check if worktree for this branch already exists
    set -l existing_worktree (__gw_find_worktree_by_branch "$branch_name")
    if test $status -eq 0
        __gw_err "Error: Worktree for branch '"$branch_name"' already exists at:" $existing_worktree
        return 1
    end

    set -l relative_path (__gw_get_relative_path_from_git_root)
    or return 1

    set -l worktree_path (__gw_compute_worktree_path "$branch_name")
    or return 1

    # Create worktree from current branch (no new branch)
    git worktree add "$worktree_path" "$branch_name"
    or return 1

    __gw_post_create_worktree "$worktree_path" "$relative_path"

    __gw_info "Created worktree for existing branch '"$branch_name"'"
end

function __gw_create_worktree
    set -l branch_name $argv[1]

    set -l relative_path (__gw_get_relative_path_from_git_root)
    or return 1

    set -l worktree_path (__gw_compute_worktree_path "$branch_name")
    or return 1

    git worktree add "$worktree_path" -b "$branch_name"
    or return 1

    __gw_post_create_worktree "$worktree_path" "$relative_path"
end

function __gw_restore_branch
    set -l branch_name $argv[1]

    set -l relative_path (__gw_get_relative_path_from_git_root)
    or return 1

    set -l worktree_path (__gw_compute_worktree_path "$branch_name")
    or return 1

    git worktree add "$worktree_path" "$branch_name"
    or return 1

    __gw_post_create_worktree "$worktree_path" "$relative_path"
end

function __gw_switch_to_path
    set -l path $argv[1]
    set -l current_worktree (__gw_get_current_worktree_path)
    set -l current_branch ""
    set -l target_branch ""

    # Get current branch
    if test -n "$current_worktree"
        set current_branch (__gw_get_branch_from_worktree "$current_worktree")
    end

    # Get target branch
    set target_branch (__gw_get_branch_from_worktree "$path")

    # Get relative path and navigate to it
    set -l relative_path (__gw_get_relative_path_from_git_root)
    __gw_navigate_to_relative_path "$path" "$relative_path"

    # Show movement information
    if test -n "$current_branch" -a -n "$target_branch"
        __gw_info "Switched from ["$current_branch"] to ["$target_branch"]"
    else
        __gw_info "Switched to worktree:" $path
    end
end

function __gw_switch_worktree
    __gw_execute_with_selected_worktree "Select worktree: " __gw_switch_to_path true
end

function __gw_remove_worktree
    set -l force_flag $argv[1]
    __gw_execute_with_selected_worktree "Select worktree to remove: " "__gw_remove_worktree_at_path_wrapper $force_flag" true
end

function __gw_remove_worktree_at_path_wrapper
    set -l force_flag $argv[1]
    set -l path $argv[2]
    __gw_remove_worktree_at_path "$path" "$force_flag"
end

function __gw_remove_worktree_by_branch
    set -l branch_name $argv[1]
    set -l force_flag $argv[2]

    set -l worktree_path (__gw_find_worktree_by_branch "$branch_name")
    if test $status -ne 0
        __gw_err "Error: No worktree found for branch:" $branch_name
        return 1
    end

    # Check if trying to remove current worktree
    set -l current_worktree (__gw_get_current_worktree_path)
    if test -n "$current_worktree" -a "$worktree_path" = "$current_worktree"
        __gw_err "Error: Cannot remove current worktree for branch:" $branch_name
        __gw_info "Please switch to a different worktree first"
        return 1
    end

    __gw_remove_worktree_at_path "$worktree_path" "$force_flag"
end

function __gw_remove_multiple_worktrees
    set -l force_flag $argv[1]
    set -l branch_names $argv[2..-1]
    set -l success_count 0
    set -l failed_branches

    for branch_name in $branch_names
        echo "Processing branch: $branch_name"
        if __gw_remove_worktree_by_branch "$branch_name" "$force_flag"
            set success_count (math $success_count + 1)
            echo "✓ Successfully removed worktree for branch: $branch_name"
        else
            set -a failed_branches "$branch_name"
            echo "✗ Failed to remove worktree for branch: $branch_name"
        end
        echo ""
    end

    echo "Summary:"
    echo "  Successfully removed: $success_count worktree(s)"
    if test (count $failed_branches) -gt 0
        echo "  Failed branches: "(string join ", " $failed_branches)
        return 1
    end
end

function __gw_find_worktree_by_branch
    set -l target_branch $argv[1]

    set -l worktree_info (git worktree list --porcelain)
    or return 1

    set -l current_path ""
    set -l current_branch ""

    for line in $worktree_info
        switch $line
            case "worktree *"
                set current_path (string replace "worktree " "" "$line")
            case "branch *"
                set current_branch (string replace "branch refs/heads/" "" "$line")
            case "HEAD *"
                set current_branch HEAD
            case ""
                if test -n "$current_path" -a "$current_branch" = "$target_branch"
                    echo "$current_path"
                    return 0
                end
                set current_path ""
                set current_branch ""
        end
    end

    if test -n "$current_path" -a "$current_branch" = "$target_branch"
        echo "$current_path"
        return 0
    end

    return 1
end

function __gw_switch_to_branch
    set -l branch_name $argv[1]

    set -l worktree_path (__gw_find_worktree_by_branch "$branch_name")
    or begin
        echo "Error: No worktree found for branch: $branch_name"
        return 1
    end

    set -l current_worktree (__gw_get_current_worktree_path)
    set -l current_branch ""

    # Get current branch
    if test -n "$current_worktree"
        set current_branch (__gw_get_branch_from_worktree "$current_worktree")
    end

    set -l relative_path (__gw_get_relative_path_from_git_root)
    __gw_navigate_to_relative_path "$worktree_path" "$relative_path"

    # Show movement information
    if test -n "$current_branch"
        echo "Switched from [$current_branch] to [$branch_name]"
    else
        echo "Switched to worktree for branch: $branch_name"
    end
end

function __gw_get_current_worktree_path
    set -l current_dir (pwd)
    set -l worktree_info (git worktree list --porcelain 2>/dev/null)

    if test -z "$worktree_info"
        return 1
    end

    set -l current_path ""
    set -l best_match ""
    set -l best_match_length 0

    for line in $worktree_info
        switch $line
            case "worktree *"
                set current_path (string replace "worktree " "" "$line")
                # Check if current directory is within this worktree
                if string match -q "$current_path*" "$current_dir"
                    set -l match_length (string length "$current_path")
                    # Find the most specific (longest) matching path
                    if test $match_length -gt $best_match_length
                        set best_match "$current_path"
                        set best_match_length $match_length
                    end
                end
            case ""
                set current_path ""
        end
    end

    # Handle the last entry (no trailing empty line)
    if test -n "$current_path"
        if string match -q "$current_path*" "$current_dir"
            set -l match_length (string length "$current_path")
            if test $match_length -gt $best_match_length
                set best_match "$current_path"
                set best_match_length $match_length
            end
        end
    end

    if test -n "$best_match"
        echo "$best_match"
        return 0
    end

    return 1
end

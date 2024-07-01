# synclib

A small Go tool for creating symbolic links

Created out of infuriating difficulty of creating symbolic links on windows

## Custom syntax

The tool works with "instructions" that describe symbolic links

They are, in any form, \<source>,\<destination>,\<force?>

For example:
`sync this,that`

It supports input of these instructions through:

- Stdin
	- `echo "this,that" | sync`
- Run arguments
	- `sync this,that foo,bar "foo 2","C:/bar"`
- Files
	- `sync -f <file>`
	- Where the file contains instructions, one instruction per line
- Directories
	- `sync -r <directory>`
	- This mode will look for "sync" files recursively in directories and run their instructions

## Use case

I have a lot of folders (documents, projects, configurations) backed up via Seafile and to have the software using those folders find them at their usual location I'm creating soft symbolic links from the seafile drive to their original location

It would be problematic to have to redo all (or some part) of these symlinks when reinstalling the OS or having something somewhere explode (say software uninstalled) so I have all the instructions in sync files in individual folders in the seafile drive

Which means I can easily back up my configuration and `sync -r ~/Seafile` to symlink it where it belongs
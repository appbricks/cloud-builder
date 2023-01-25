#!/bin/bash

root_dir=$(pwd)
home_dir=${root_dir}/cookbook

set -e

usage () {
  echo -e "\nUSAGE: build-cookbook.sh -r|--recipe <RECIPE REPO PATH> \\"
  echo -e "    [-b|--git-branch <GIT_BRANCH_NAME>] \\"
  echo -e "    [-n|--name <RECIPE NAME>] [-i|--iaas <TARGET IAAS>] \\"
  echo -e "    [--cookbook-name <COOKBOOK NAME>] \\"
  echo -e "    [--cookbook-desc <COOKBOOK DESCRIPTION>] \\"
  echo -e "    [--cookbook-version <COOKBOOK VERSION>] \\"
  echo -e "    [-o|--os-name <TARGET OS>] [-a|--os-arch <TARGET OS ARCH>] \\"
  echo -e "    [-d|--dest-dir <COOKBOOK_DEST_DIR>] \\"
  echo -e "    [-s|--single] [-c|--clean]] [-v|--verbose]\n"
  echo -e "    This utility script packages the terraform recipes or distribution with the service."
  echo -e "    The Terraform recipe should exist under the given repo path within a folder having a"
  echo -e "    <recipe name>/<iaas> folder. The 'recipe', 'name' and 'iaas' options are all required"
  echo -e "    when adding a recipe repo to the distribution.\n"
  echo -e "    -r|--recipe           <RECIPE REPO PATH>     (required) The path to the git repo."
  echo -e "                                                 i.e https://github.com/<user>/<repo>/<path>."
  echo -e "    -b|--git-branch       <GIT_BRANCH_NAME>      The branch or tag of the git repository. Default is \"master\"."
  echo -e "    -n|--name             <RECIPE NAME>          The name of the recipe"
  echo -e "    -i|--iaas             <TARGET IAAS>          The target IaaS of this recipe."
  echo -e "       --cookbook-name    <COOKBOOK NAME>        The cookbook name"
  echo -e "       --cookbook-desc    <COOKBOOK DESCRIPTION> A description for the cookbook"
  echo -e "       --cookbook-version <COOKBOOK VERSION>     The version of the cookbook"
  echo -e "    -o|--os-name          <TARGET OS>            The target OS for which recipe providers should be download."
  echo -e "                                                 Should be one of \"darwin\", \"linux\" or \"windows\"."
  echo -e "    -a|--os-arch          <TARGET OS ARCH>       The target OS architecture."
  echo -e "                                                 Should be one of \"386\", \"amd64\", \"arm\", \"arm64\"."
  echo -e "    -d|--dest-dir         <COOKBOOK_DEST_DIR>    The cookbook destination directory."
  echo -e "                                                 Default is <CURR_DIR>/cookbook/dist."
  echo -e "    -t|--template-only                           Add only templates and template plugin dependencies to archive"
  echo -e "    -s|--single                                  Only the recipe indicated shoud be added"
  echo -e "    -c|--clean                                   Clean build before proceeding"
  echo -e "    -v|--verbose                                 Trace shell execution"
}

recipe_git_branch_or_tag=master
recipe_iaas=""
target_os=$(go env GOOS)
target_arch=$(go env GOARCH)

cookbook_dest_dir=${HOME_DIR:-${root_dir}/cookbook}/dist

if [[ $# -eq 0 ]]; then
  usage
  exit 1
fi

while [[ $# -gt 0 ]]; do

  case "$1" in
    '-?'|--help|help)
      usage
      exit 0
      ;;
    -r|--recipe)
      recipe_project_uri=$2
      has_recipe=1
      shift
      ;;
    -b|--git-branch)
      recipe_git_branch_or_tag=$2
      shift
      ;;
    -n|--name)
      recipe_name=$2
      shift
      ;;
    -i|--iaas)
      recipe_iaas=$2
      shift
      ;;
    --cookbook-name)
      cookbook_name=$2
      shift
      ;;
    --cookbook-desc)
      cookbook_desc=$2
      shift
      ;;
    --cookbook-version)
      cookbook_version=$2
      shift
      ;;
    -o|--os-name)
      target_os=$2
      [[ -n $(echo ":darwin:linux:windows:" | grep ":$target_os:") ]] || (
        echo "ERROR! Only OS types darwin, linux or windows are supported.";
        exit 1;
      )
      shift
      ;;
    -a|--os-arch)
      target_arch=$2
      [[ -n $(echo ":386:amd64:arm:arm64:" | grep ":$target_arch:") ]] || (
        echo "ERROR! Only OS archs 386, amd64, arm or arm64 are supported.";
        exit 1;
      )
      shift
      ;;
    -d|--dest-dir)
      cookbook_dest_dir=$2
      shift
      ;;
    -t|--template-only)
      template_only=1
      ;;
    -s|--single)
      single=1
      ;;
    -c|--clean)
      clean=1
      ;;
    -v|--verbose)
      debug=1
      ;;
    *)
      usage
      exit 1
      ;;
  esac

  shift
done

[[ -z $debug ]] || set -x

if [[ -z $recipe_project_uri ]]; then
  usage
  exit 1
fi

current_os=$(go env GOOS)
current_arch=$(go env GOARCH)

build_dir=${root_dir}/.build/cookbook
recipe_repo_dir=${build_dir}/repos
bin_dir=${build_dir}/bin
plugin_mirror_dir=${build_dir}/bin/plugins
dist_dir=${build_dir}/dist/${target_os}_${target_arch}
dest_dist_dir=${HOME_DIR:-$home_dir}/dist
cookbook_bin_dir=${dist_dir}/bin
cookbook_plugins_dir=${dist_dir}/bin/plugins
cookbook_dist_zip=${build_dir}/dist/cookbook-${target_os}_${target_arch}.zip

[[ -z $clean ]] || \
  (rm -fr $dist_dir && rm -f $cookbook_dist_zip)

terraform_version=${TERRAFORM_VERSION:-1.3.6}

mkdir -p $bin_dir
terraform=${bin_dir}/terraform
if [[ ! -e $terraform ]]; then
  curl \
    -L https://releases.hashicorp.com/terraform/${terraform_version}/terraform_${terraform_version}_${current_os}_${current_arch}.zip \
    -o ${bin_dir}/terraform.zip

  pushd $bin_dir
  unzip -o terraform.zip
  rm -f terraform.zip
  popd
fi

mkdir -p $cookbook_plugins_dir
if [[ $target_os == windows ]]; then
  cookbook_terraform_binary=${cookbook_bin_dir}/terraform.exe
else
  cookbook_terraform_binary=${cookbook_bin_dir}/terraform
fi
if [[ ! -e $cookbook_terraform_binary ]]; then
  curl \
    -L https://releases.hashicorp.com/terraform/${terraform_version}/terraform_${terraform_version}_${target_os}_${target_arch}.zip \
    -o ${cookbook_bin_dir}/terraform.zip

  pushd $cookbook_bin_dir
  unzip -o terraform.zip
  rm -f terraform.zip
  popd
fi

cookbook_recipes_dir=${dist_dir}/recipes
[[ -z $single ]] || rm -fr $cookbook_recipes_dir

if [[ $recipe_project_uri == https://* ]]; then
  url_path=${recipe_project_uri#https://*}
elif [[ $recipe_project_uri == http://* ]]; then
  url_path=${recipe_project_uri#http://*}
elif [[ -e $recipe_project_uri ]]; then
  repo_path=$recipe_project_uri
else
  echo "The URI $recipe_project_uri must be a URL to a git repo path (i.e. https://github.com/repo/path) or a local system path."
  exit 1
fi

if [[ -n $url_path ]]; then
  git_server=${url_path%%/*}
  repo_path=${url_path#*/}

  if [[ $git_server == http* || $repo_path == http* ]]; then
    echo "Unable to determine repo path. Please provide a git server name to allow the path to parsed properly."
    exit 1
  fi
fi

if [[ -n $git_server ]]; then

  repo_org=${repo_path%%/*}
  repo_org_path=${repo_path#*/}
  repo_name=${repo_org_path%%/*}
  repo_folder=${repo_path#$repo_org/$repo_name/}

  if [[ -e ${recipe_repo_dir}/${repo_name} ]]; then
    pushd ${recipe_repo_dir}/${repo_name}
    git checkout $recipe_git_branch_or_tag

    # do not pull tags
    [[ $recipe_git_branch_or_tag =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || git pull
  else
    git clone https://${git_server}/${repo_org}/${repo_name} ${recipe_repo_dir}/${repo_name}
    pushd ${recipe_repo_dir}/${repo_name}
    git checkout $recipe_git_branch_or_tag
  fi
  popd
else
  
  repo_name=$(basename $root_dir)
  repo_folder=${repo_path#$root_dir/*}

  mkdir -p $(dirname ${recipe_repo_dir}/${repo_name}/${repo_folder})
  rsync -qavr -L -P $repo_path/* ${recipe_repo_dir}/${repo_name}/${repo_folder}
fi

[[ -n $cookook_name ]] || cookook_name=$repo_name

if [[ -z $recipe_name ]]; then
  repo_list=$(ls ${recipe_repo_dir})
else
  repo_list=$repo_name
fi

for repo in $(ls ${recipe_repo_dir}); do
  for recipe in $(ls ${recipe_repo_dir}/${repo}/${repo_folder}); do
    [[ -z $recipe_name || $recipe_name == $recipe ]] || continue

    iaas_list=${recipe_iaas:-$(ls ${recipe_repo_dir}/${repo_name}/${repo_folder}/${recipe})}
    for iaas in $iaas_list; do
      echo "Adding iaas \"${iaas}\" for recipe \"${repo}/${recipe}\"..."

      recipe_folder=${recipe_repo_dir}/${repo_name}/${repo_folder}/${recipe}/${iaas}
      if [[ ! -e $recipe_folder ]]; then
        echo -e "\nERROR! Recipe folder '$recipe_folder' does not exist.\n"
        exit 1
      fi

      set +e
      ls $recipe_folder/*.tf >/dev/null 2>&1
      if [[ $? -ne 0 ]]; then
        echo -e "\nERROR! No Terraform templates found at '$recipe_folder'.\n"
        exit 1
      fi
      set -e
      cd

      # initialize terraform templates in order to
      # download the dependent providers and modules
      pushd $recipe_folder
      $terraform init -backend=false
      rm .terraform.lock.hcl
      $terraform providers lock -platform ${target_os}_${target_arch}
      rm -fr $plugin_mirror_dir
      $terraform providers mirror -platform ${target_os}_${target_arch} $plugin_mirror_dir
      popd

      # consolidate terraform providers to
      # the distribution's provider folder
      # downloading os specific binaries
      # if os is different to build os
      for f in $(find $plugin_mirror_dir -name "terraform-provider-*_${target_os}_${target_arch}.zip" -print); do
        abs_dir_path=$(dirname $f)
        provider_filename=$(basename $f)
        provider_version=$(echo "$provider_filename" | sed -e "s|.*_\([0-9]*\(\.[0-9]*\)*\)_${target_os}_${target_arch}.zip|\1|")
        provider_path=${abs_dir_path#${plugin_mirror_dir}/*}
        provider_dist_path=${cookbook_plugins_dir}/${provider_path}/${provider_version}/${target_os}_${target_arch}
        mkdir -p $provider_dist_path

        mv $f $provider_dist_path
        pushd $provider_dist_path
        unzip -o $provider_filename
        rm $provider_filename
        popd
      done

      rm -fr ${cookbook_recipes_dir}/${recipe}/${iaas}
      mkdir -p ${cookbook_recipes_dir}/${recipe}/${iaas}
      cp -RLp $recipe_folder ${cookbook_recipes_dir}/${recipe}
      rm -f ${cookbook_recipes_dir}/${recipe}/${iaas}/.terraform/terraform.tfstate
      rm -fr ${cookbook_recipes_dir}/${recipe}/${iaas}/.terraform/providers
    done

  done
done

pushd ${dist_dir}

terraform_version=$($terraform version | awk '/^Terraform v/{ print substr($2, 2) }')

cat << ---EOF > METADATA
---
cookbook-name: '$cookbook_name'
cookbook-version: '$cookbook_version'
description: '$cookbook_desc'
terraform-version: '$terraform_version'
target-os-name: '$target_os'
target-os-arch: '$target_arch'
---EOF

if [[ -n $template_only ]]; then
  zip -ur $cookbook_dist_zip . -x "*.git*" -x "bin/terraform"
else
  zip -ur $cookbook_dist_zip . -x "*.git*"
fi
popd

if [[ -n $cookbook_dest_dir ]]; then
  [[ -z $clean ]] || rm -fr $cookbook_dest_dir

  mkdir -p ${cookbook_dest_dir}
  rm -f ${cookbook_dest_dir}/cookbook.zip
  cp $cookbook_dist_zip ${cookbook_dest_dir}/cookbook.zip

  if [[ $current_os == linux ]]; then
    stat -t -c "%Y" ${cookbook_dest_dir}/cookbook.zip > ${cookbook_dest_dir}/cookbook-mod-time
  elif [[ $current_os == darwin ]]; then
    stat -t "%s" -f "%Sm" ${cookbook_dest_dir}/cookbook.zip > ${cookbook_dest_dir}/cookbook-mod-time
  else
    echo -e "\nERROR! Unable to get the modification timestamp of '${cookbook_dest_dir}/cookbook.zip'.\n"
    exit 1
  fi
fi

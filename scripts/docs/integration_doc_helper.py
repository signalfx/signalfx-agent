import os
import re

from pathlib import Path

SCRIPT_DIR = Path(__file__).parent.resolve()
AGENT_ROOT = SCRIPT_DIR / "../.."
AGENT_DOCS = AGENT_ROOT / "docs"

INTEGRATIONS_REPO = Path(os.environ.get("INTEGRATIONS_REPO") or (AGENT_ROOT / "../integrations")).resolve()
INTEGRATIONS_REPO_SMART_AGENT_DIR = INTEGRATIONS_REPO / "signalfx-agent"
INTEGRATIONS_DOC_TEMPLATE_DIR = AGENT_ROOT / "scripts/docs/templates/"
RELATIVE_LINK_PATTERN = re.compile(r"(\[.*?\])\((\.\.?/.+?)\)", flags=re.DOTALL)

AUTO_GENERATION_TEMPLATE = """
<!--- Generated by to-integrations-repo script in Smart Agent repo, DO NOT MODIFY HERE --->
"""


def fixup_relative_paths(absolute_path, content):
    """
    Only works for links within the specified absolute path
    """
    return RELATIVE_LINK_PATTERN.sub(absolute_path, content)


def fixup_relative_monitor_paths(content):
    """
    Replaces relative links within the scope of monitor
    docs with respective github links
    """
    return fixup_relative_paths(r"\1(https://github.com/signalfx/signalfx-agent/tree/main/docs/monitors/\2)", content)


def fixup_relative_agent_doc_paths(content):
    """
    Replaces relative links within the scope of agent docs
    with respective product-docs links
    """
    return fixup_relative_paths(r"\1(https://docs.signalfx.com/en/latest/integrations/agent/\2)", content)


# names of directories and files that need to be eventually
# surfaced in product-docs. make use of this map to process
# and sync only required docs to the integrations repo. Note
# that all files from directories specified will be synced to
# the integrations repo. The below dictionary is populated from
# https://github.com/signalfx/signalfx-agent/tree/main/docs/
REQUIRED_AGENT_DOCS = {
    "directories": ["monitors", "observers"],
    "md_files": [
        "agent-install-awsecs",
        "agent-install-config-mgmt",
        "agent-install-packages",
        "agent-install-standalone-linux",
        "agent-install-standalone-windows",
        "agent-k8s-install-helm",
        "agent-k8s-install-kubectl",
        "quick-install",
        "config-schema",
        "observer-config",
        "monitor-config",
        "auto-discovery",
        "filtering",
        "remote-config",
        "windows",
        "faq",
        "legacy-filtering",
        "deb-rpm-repo-migration",
        "smartagent-deprecation-notice",
    ],
}

AGENT_README_HEADER = """
# ![](https://github.com/signalfx/integrations/blob/master/signalfx-agent/img/integration_smartagent.png) SignalFx Smart Agent

"""

MARKDOWN_LINK_PATTERN = re.compile(r"(\(https://docs.signalfx.com/en/latest/integrations/agent/.+?\.)md(.*?\))")
MARKDOWN_SUBSECTION_LINK_PATTERN = re.compile(r"(\[.*?\]\(\.\.?/.+?\.)md(\#.+?\))", flags=re.DOTALL)
SFX_APP_LINK_PATTERN = re.compile("\[\]\(sfx_link:.+?\)")


def convert_markdowns_to_htmls(content):
    return re.sub(MARKDOWN_LINK_PATTERN, r"\1html\2", content)


def convert_markdown_subsections_to_htmls(content):
    return re.sub(MARKDOWN_SUBSECTION_LINK_PATTERN, r"\1html\2", content)


def fixup_headers_in_agent_readme(content):
    # This is a hack in place for the README to surfaced in desired
    # manner on the tile, so that the magic comment has an invisible
    # header to use
    content = content.replace("# Quick Install", "## <!-- -->")
    content = content.replace(" - [Concepts](#concepts)", "")
    content = content.replace(" - [Installation](#installation)", "")
    content = content.replace("## Concepts", "### Concepts")
    content = content.replace("### Monitors", "#### Monitors")
    content = content.replace("### Observers", "#### Observers")
    content = content.replace("### Writer", "#### Writer")

    return content


def remove_sfx_app_links(content):
    return re.sub(SFX_APP_LINK_PATTERN, r"", content)


def fixup_moved_links(content):
    content = content.replace("(./observer-config.md)", "(./observers/_observer-config.md)")
    content = content.replace("(./monitor-config.md)", "(./monitors/_monitor-config.md)")

    return content


def sync_agent_quick_install():
    """
    Construct README for SignalFx Agent from quick-install.md file here:
    https://github.com/signalfx/signalfx-agent/tree/main/docs
    """

    target_path = INTEGRATIONS_REPO_SMART_AGENT_DIR / "README.md"
    smart_agent_quick_install_path = AGENT_DOCS / "quick-install.md"

    print("Constructing Agent README")

    target_path.write_text(
        convert_markdown_subsections_to_htmls(
            AUTO_GENERATION_TEMPLATE
            + convert_markdowns_to_htmls(
                fixup_relative_agent_doc_paths(
                    AGENT_README_HEADER
                    + fixup_headers_in_agent_readme(
                        fixup_moved_links(smart_agent_quick_install_path.read_text(encoding="utf-8"))
                    )
                )
            )
        ),
        encoding="utf-8",
    )


PRODUCT_DOCS_REPO = "agent_docs"


def sync_agent_docs():
    """
    Sync Agent docs from here:
    https://github.com/signalfx/signalfx-agent/tree/main/docs
    """
    target_dir_parent_path = INTEGRATIONS_REPO_SMART_AGENT_DIR / PRODUCT_DOCS_REPO
    target_dir_parent_path.mkdir(parents=True, exist_ok=True)

    for dir in REQUIRED_AGENT_DOCS["directories"]:
        full_dir_path = AGENT_DOCS / dir
        assert full_dir_path.is_dir(), full_dir_path

        target_dir_path = target_dir_parent_path / dir
        target_dir_path.mkdir(parents=True, exist_ok=True)
        assert target_dir_path.is_dir(), target_dir_path

        for full_file_path in full_dir_path.iterdir():
            sync_markdown_files(full_file_path)

    for file in REQUIRED_AGENT_DOCS["md_files"]:
        full_file_path = AGENT_DOCS / ("%s.md" % file)
        sync_markdown_files(full_file_path)


def sync_markdown_files(source_file_path):
    assert source_file_path.is_file(), source_file_path

    relative_path = source_file_path.relative_to(AGENT_DOCS)
    target_path = INTEGRATIONS_REPO_SMART_AGENT_DIR / PRODUCT_DOCS_REPO / relative_path

    print(f"Syncing Agent docs: {str(relative_path)}")

    target_path.write_text(AUTO_GENERATION_TEMPLATE + remove_sfx_app_links(
        convert_markdown_subsections_to_htmls(source_file_path.read_text(encoding="utf-8"))), encoding="utf-8", )


def sync_agent_info():
    sync_agent_quick_install()
    sync_agent_docs()

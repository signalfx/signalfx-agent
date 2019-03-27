from setuptools import find_packages, setup

# yapf: disable
setup(
    name="sfxpython",
    version="0.2.0",
    packages=find_packages(),
    install_requires=[
        'ujson==1.35',
        'typing==3.6.4',
    ],
)

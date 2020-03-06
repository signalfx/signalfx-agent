from setuptools import find_packages, setup

# yapf: disable
setup(
    name="sfxpython",
    version="0.3.0",
    author="SignalFx, Inc",
    description="Python packages used by the Python extension mechanism of the SignalFx Smart Agent",
    url="https://github.com/signalfx/signalfx-agent",
    packages=find_packages(),
    install_requires=[
        'ujson==1.35',
        'typing==3.6.4',
    ],
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: Apache Software License",
    ]
)

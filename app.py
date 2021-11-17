#!/usr/bin/env python3
import os

from aws_cdk import core

from htsget_stack import HtsgetStack

account_id = os.environ.get("CDK_DEFAULT_ACCOUNT")
aws_region = os.environ.get("CDK_DEFAULT_REGION")
aws_env = {"account": account_id, "region": aws_region}

# we use this string as a way of labelling artifacts.. i.e. stack ids, descriptions in security groups etc
UNIQUE_NAMESPACE = "Htsget"

app = core.App()

HtsgetStack(
    app,
    UNIQUE_NAMESPACE,
    stack_name=UNIQUE_NAMESPACE.lower(),
    # the dns prefix that is used for the ALB .. i.e. <dns_record_name>.dev.umccr.org
    dns_record_name="htsget-apse2",
    env=aws_env,
    tags={
        "Stack": UNIQUE_NAMESPACE,
        "Creator": "cdk",
        "Environment": account_id,
    }
)

app.synth()

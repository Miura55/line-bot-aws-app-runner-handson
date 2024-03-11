import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { Repository } from 'aws-cdk-lib/aws-ecr';
import { Role, ServicePrincipal, ManagedPolicy } from 'aws-cdk-lib/aws-iam';
import * as apprunner from '@aws-cdk/aws-apprunner-alpha';
import { Secret } from 'aws-cdk-lib/aws-secretsmanager';

interface AppStackProps extends cdk.StackProps {
  ecrRepository: Repository;
}

export class AppStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: AppStackProps) {
    super(scope, id, props);

    const ecrAccessRole = new Role(this, 'EcrAccessRole', {
      assumedBy: new ServicePrincipal('build.apprunner.amazonaws.com'),
      roleName: 'HandsonAppRunnerECRAccessRole',
    });

    ecrAccessRole.addManagedPolicy(
      ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSAppRunnerServicePolicyForECRAccess')
    );

    const instaceRole = new Role(this, 'InstanceRole', {
      assumedBy: new ServicePrincipal('tasks.apprunner.amazonaws.com'),
      roleName: 'HandsonAppRunnerInstanceRole',
    });

    instaceRole.addManagedPolicy(
      ManagedPolicy.fromAwsManagedPolicyName('AmazonSSMReadOnlyAccess')
    );

    const apprunnerService = new apprunner.Service(this, 'AppRunnerService', {
      source: apprunner.Source.fromEcr({
        repository: props.ecrRepository,
        tagOrDigest: 'latest',
        imageConfiguration: {
          port: 8080,
          environmentSecrets: {
            CHANNEL_SECRET: apprunner.Secret.fromSecretsManager(Secret.fromSecretPartialArn(this, 'linebot-apprunner-handson/CHANNEL_SECRET', `arn:aws:ssm:ap-northeast-1:${this.account}:parameter/linebot-apprunner-handson/CHANNEL_SECRET`)),
            CHANNEL_TOKEN: apprunner.Secret.fromSecretsManager(Secret.fromSecretPartialArn(this, 'linebot-apprunner-handson/CHANNEL_TOKEN', `arn:aws:ssm:ap-northeast-1:${this.account}:parameter/linebot-apprunner-handson/CHANNEL_TOKEN`)),
          }
        }
      }),
      accessRole: ecrAccessRole,
      instanceRole: instaceRole,
      serviceName: 'line-bot-hands-on',
      autoDeploymentsEnabled: true,
      healthCheck: apprunner.HealthCheck.http({
        path: '/health',
        healthyThreshold: 5,
        unhealthyThreshold: 10,
        interval: cdk.Duration.seconds(10),
        timeout: cdk.Duration.seconds(10),
      }),
    })

    new cdk.CfnOutput(this, 'AppRunnerServiceUrl', {
      value: apprunnerService.serviceUrl!,
    });
  }
}

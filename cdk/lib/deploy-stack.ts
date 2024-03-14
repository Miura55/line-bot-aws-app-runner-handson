import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { Repository } from 'aws-cdk-lib/aws-ecr';

export class DeployStack extends cdk.Stack {
  public readonly ecrRepository: Repository;

  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    this.ecrRepository = new Repository(this, 'EcrRepository', {
      repositoryName: 'line-bot-hands-on',
    });
  }
}

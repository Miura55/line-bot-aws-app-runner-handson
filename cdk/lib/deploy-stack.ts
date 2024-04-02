import { Stack, StackProps, RemovalPolicy } from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { Repository } from 'aws-cdk-lib/aws-ecr';

export class DeployStack extends Stack {
  public readonly ecrRepository: Repository;

  constructor(scope: Construct, id: string, props?: StackProps) {
    super(scope, id, props);

    this.ecrRepository = new Repository(this, 'EcrRepository', {
      repositoryName: `line-bot-hands-on-${this.account}`,
      autoDeleteImages: true,
      removalPolicy: RemovalPolicy.DESTROY,
    });
    this.ecrRepository.applyRemovalPolicy(RemovalPolicy.DESTROY);
  }
}

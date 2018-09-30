package decorator

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	sparta "github.com/mweagle/Sparta"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"
)

// CloudFrontSiteDistributionDecorator returns a ServiceDecoratorHookHandler
// function that provisions a CloudFront distribution whose origin
// is the supplied S3Site bucket
func CloudFrontSiteDistributionDecorator(s3Site *sparta.S3Site,
	subdomain string, domainName string) sparta.ServiceDecoratorHookHandler {
	bucketName := domainName
	if subdomain != "" {
		bucketName = fmt.Sprintf("%s.%s", subdomain, domainName)
	}
	s3Site.BucketName = gocf.String(bucketName)

	// Setup the CF distro
	distroDecorator := func(context map[string]interface{},
		serviceName string,
		template *gocf.Template,
		S3Bucket string,
		buildID string,
		awsSession *session.Session,
		noop bool,
		logger *logrus.Logger) error {
		dnsRecordResourceName := sparta.CloudFormationResourceName("DNSRecord",
			"DNSRecord")
		cloudFrontDistroResourceName := sparta.CloudFormationResourceName("CloudFrontDistro",
			"CloudFrontDistro")

		// Use the HostedZoneName to create the record
		hostedZoneName := fmt.Sprintf("%s.", domainName)
		dnsRecordResource := &gocf.Route53RecordSet{
			// // Zone for the mweagle.io
			HostedZoneName: gocf.String(hostedZoneName),
			Name:           gocf.String(bucketName),
			Type:           gocf.String("A"),
			AliasTarget: &gocf.Route53RecordSetAliasTarget{
				// This HostedZoneID value is required...
				HostedZoneID: gocf.String("Z2FDTNDATAQYW2"),
				DNSName:      gocf.GetAtt(cloudFrontDistroResourceName, "DomainName"),
			},
		}
		template.AddResource(dnsRecordResourceName, dnsRecordResource)
		// IndexDocument
		indexDocument := gocf.String("index.html")
		if s3Site.WebsiteConfiguration != nil &&
			s3Site.WebsiteConfiguration.IndexDocument != nil &&
			s3Site.WebsiteConfiguration.IndexDocument.Suffix != nil {
			indexDocument = gocf.String(*s3Site.WebsiteConfiguration.IndexDocument.Suffix)
		}
		// Add the distro...
		distroConfig := &gocf.CloudFrontDistributionDistributionConfig{
			Aliases:           gocf.StringList(s3Site.BucketName),
			DefaultRootObject: indexDocument,
			Origins: &gocf.CloudFrontDistributionOriginList{
				gocf.CloudFrontDistributionOrigin{
					DomainName:     gocf.GetAtt(s3Site.CloudFormationS3ResourceName(), "DomainName"),
					ID:             gocf.String("S3Origin"),
					S3OriginConfig: &gocf.CloudFrontDistributionS3OriginConfig{},
				},
			},
			Enabled: gocf.Bool(true),
			DefaultCacheBehavior: &gocf.CloudFrontDistributionDefaultCacheBehavior{
				ForwardedValues: &gocf.CloudFrontDistributionForwardedValues{
					QueryString: gocf.Bool(false),
				},
				TargetOriginID:       gocf.String("S3Origin"),
				ViewerProtocolPolicy: gocf.String("allow-all"),
			},
		}
		cloudfrontDistro := &gocf.CloudFrontDistribution{
			DistributionConfig: distroConfig,
		}
		template.AddResource(cloudFrontDistroResourceName, cloudfrontDistro)
		return nil
	}
	return sparta.ServiceDecoratorHookFunc(distroDecorator)
}
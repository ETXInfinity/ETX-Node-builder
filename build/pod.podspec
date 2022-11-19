Pod::Spec.new do |spec|
  spec.name         = 'Getx'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ETX/go-ETX'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS ETX Client'
  spec.source       = { :git => 'https://github.com/ETX/go-ETX.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Getx.framework'

	spec.prepare_command = <<-CMD
    curl https://getxstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Getx.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end

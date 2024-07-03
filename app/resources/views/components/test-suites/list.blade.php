@props(['title' => 'Available Tests', 'suites' => []])

<div class="list-interactive">
    @foreach($suites as $repositoryTitle => $repository) 
        <div class="list-repository list">
            <div class="title expand"> 
                {{ $repositoryTitle }}
            </div>

            @foreach($repository as $serviceTitle => $service)
                <div class="list-service list">
                    <div class="title expand"> 
                        {{ $serviceTitle }}
                    </div>

                    @foreach($service as $suiteTitle => $suite)
                        <div class="list-suite list">
                            <div class="title expand"> 
                                {{ $suiteTitle }}
                            </div>
                            <div class="list-test list">
                                @foreach($suite as $test)
                                    <div class="test">
                                        {{ $test['test_name'] }}
                                    </div>
                                @endforeach
                            </div>
                        </div>
                    @endforeach
                
                </div>
            @endforeach
        
        </div>
    @endforeach
</div>



@props(['select' => true, 'title' => 'Select Tests', 'suites' => []])
<table class="select">
    <thead>
        <tr>
            <th colspan="2">{{$title}}</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        @foreach($suites as $path => $suite)
        <tr>
            <td>
                @if ($select)
                <div>
                    <label for="selector-suite-{{ $path }}" class="checkbox suite">
                        <input class="suite-selector" type="checkbox" id="selector-suite-{{ $path }}" value="{{$path}}">
                        <span class="checkmark"></span>
                    </label>
                </div>
                @endif
            </td>
            <td>{{getTestSuiteName($suite[0])}}</td>
            <td><span class="expand" data-target="selector-suite-tests-{{ $path }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $path }}" class="collapsible">
            <td colspan="3">
                <table>
                    @foreach ($suite as $test)
                    <tr>
                        <td>
                            @if ($select)
                            <div>
                                <label for="selector-test-{{ $path.$test['test_name'] }}" class="checkbox suite">
                                    <input class="test-selector" type="checkbox" data-parent="{{ $path }}" id="selector-test-{{ $path.$test['test_name'] }}" name="tests[]" value="{{ json_encode($test) }}">
                                    <span class="checkmark"></span>
                                </label>
                            </div>
                            @endif
                        </td>
                        <td>{{$test['test_name']}}</td>
                    </tr>
                    @endforeach
                </table>
            </td>
        </tr>
        @endforeach
    </tbody>
</table>
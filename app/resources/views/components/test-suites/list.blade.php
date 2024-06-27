@props(['title' => 'Selected Tests', 'suites' => []])
<table>
    <thead>
        <tr>
            <th colspan="2">{{$title}}</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        @foreach($suites as $path => $suite)
        <tr>
            <td>{{getTestSuiteName($suite[0])}}</td>
            <td><span class="expand" data-target="selector-suite-tests-{{ $path }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $path }}" class="collapsible">
            <td colspan="2">
                <table>
                    @foreach ($suite as $test)
                    <tr>
                        <td>{{$test['test_name']}}</td>
                    </tr>
                    @endforeach
                </table>
            </td>
        </tr>
        @endforeach
    </tbody>
</table>
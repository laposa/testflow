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
                <div class="suite-selector">
                    <label for="selector-suite-{{ $path }}" class="checkbox suite">
                        <input type="checkbox" id="selector-suite-{{ $path }}" value="{{$path}}">
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
                            <div class="test-selector">
                                <label for="selector-test-{{ $path.$test['test_name'] }}" class="checkbox suite">
                                    <input type="checkbox" data-parent="{{ $path }}" id="selector-test-{{ $path.$test['test_name'] }}" name="tests[]" value="{{ json_encode($test) }}">
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


<script>
    document.querySelectorAll('.suite-selector input').forEach((checkbox) => {
        checkbox.addEventListener('change', (event) => {
            const suite = event.target.closest('tr').nextElementSibling;
            suite.querySelectorAll('input').forEach((test) => {
                test.checked = event.target.checked;
            });
        });
    });

    document.querySelectorAll('.test-selector input').forEach((checkbox) => {
        checkbox.addEventListener('change', (event) => {
            const suiteCheckbox = document.getElementById("selector-suite-" + event.target.dataset.parent);
            const allTests = document.getElementById("selector-suite-tests-" + event.target.dataset.parent).querySelectorAll('input');
            const checkedTests = Array.from(allTests).filter((test) => test.checked);
            suiteCheckbox.checked = checkedTests.length === allTests.length;
        });
    });
</script>
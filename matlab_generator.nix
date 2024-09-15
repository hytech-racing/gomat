{ lib, python311Packages } :

python311Packages.buildPythonApplication {
    pname = "matlab_generator";
    version = "1.0.0";

    propogatedBuildInputs = [
        python311Packages.scipy
    ];

    src = ./matlab_generator;
}
